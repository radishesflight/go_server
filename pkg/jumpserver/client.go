// Package jumpserver 提供精简版 JumpServer Core API 客户端
//
// 设计原则:
//   - 只覆盖本项目实际用到的端点,不全量实现
//   - 凭据(base_url / token / org_id)由调用方从 config.AppConfig.Jumpserver 传入
//     本 package 不读任何全局配置,保持纯函数式
//   - 返回值尽量用领域类型(Node)而不是裸 map
//
// 当前实现的端点:
//   - GET /api/v1/assets/nodes/children/tree/?assets=0&   拉资产根节点树
//
// 不做什么:
//   - 不实现 Koko SSH 通道(那是 web 终端专用,这里只跑命令用 Core API)
//   - 不做连接池(每次 NewClient 一个 http.Client,30s timeout)
package jumpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Node JMS 节点树的单个节点(zTree 风格)
//
//	{
//	  "id": "1:17",        // zTree 内部编号,不用
//	  "name": "一心堂 (4)", // → 业务项目 name(展示名,带后缀表示该节点下资产数)
//	  "pId": "1",          // 父节点 zTree id
//	  "isParent": true,
//	  "meta": {
//	    "type": "node",    // "node" = 组织节点(本字段为 "asset" 时是资产,本接口基本不返回)
//	    "data": {
//	      "id":   "52505e9b-...", // → 业务项目 code(机器名,UUID 风格)
//	      "key":  "1:17",
//	      "value":"一心堂"        // 跟 name 类似
//	    }
//	  }
//	}
type Node struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	PID      string   `json:"pId"`
	IsParent bool     `json:"isParent"`
	Meta     NodeMeta `json:"meta"`
}

// NodeMeta meta 嵌套结构
type NodeMeta struct {
	Type string   `json:"type"` // "node" | "asset" | ...
	Data NodeData `json:"data"`
}

// NodeData 节点/资产的核心数据
type NodeData struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Client JMS HTTP 客户端
type Client struct {
	baseURL string
	token   string
	orgID   string
	http    *http.Client
}

// Asset JMS 资产(/api/v1/assets/assets/ 列表项的最小子集)
//
// 实际接口返回 30+ 字段,这里只取同步需要的几个;多余字段会被 json.Unmarshal 忽略
//
//	nodes[0] 决定"这个资产属于哪个项目",存 jms_id(uuid) 和 name(展示名)
type Asset struct {
	ID        string       `json:"id"`        // jms asset uuid
	Name      string       `json:"name"`      // 资产名(展示)
	Address   string       `json:"address"`   // IP/域名
	Platform  *AssetPlat   `json:"platform"`  // 平台(Linux/Windows/...)
	Nodes     []Node       `json:"nodes"`     // 所属项目树(只取 nodes[0])
	Protocols []AssetProto `json:"protocols"` // 主要协议(取第一个)
	Comment   string       `json:"comment"`
}

type AssetPlat struct {
	Name string `json:"name"`
}

type AssetProto struct {
	Name string `json:"name"`
	Port int    `json:"port"`
}

// AssetListResp 分页响应
type AssetListResp struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`     // 下一页 URL(完整),null 表示到底
	Previous *string `json:"previous"` // 上一页 URL(完整)
	Results  []Asset `json:"results"`
}

// NewClient 构造一个 Client
//   - baseURL: e.g. "http://118.24.89.187:8081" (不要带尾斜杠,带会触发 //api/v1/... 404)
//   - token:   JMS 用户的 Token(从 JMS 个人中心拿)
//   - orgID:   组织 ID(多组织场景用,UUID 字符串)
func NewClient(baseURL, token, orgID string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		orgID:   orgID,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// GetNodeTree 拉资产根节点树
// 端点:GET {baseURL}/api/v1/assets/nodes/children/tree/?assets=0&
// assets=0 表示不展开资产(只拿节点/组织),& 末尾留个空 query
//
// 返回的节点需要调用方自行过滤:
//   - meta.type == "node" 才是组织节点
//   - meta.data.id 非空(根节点"Default"在 zTree 里 id=1 但 meta.data.id 是空字符串)
func (c *Client) GetNodeTree() ([]Node, error) {
	url := c.baseURL + "/api/v1/assets/nodes/children/tree/?assets=0&"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("X-JMS-ORG", c.orgID)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("JMS 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("JMS GET node-tree -> %d: %s", resp.StatusCode, string(body))
	}

	var nodes []Node
	if err := json.Unmarshal(body, &nodes); err != nil {
		return nil, fmt.Errorf("JMS 响应解析失败: %w (body 前 200 字符: %.200s)", err, string(body))
	}
	return nodes, nil
}

// ListAssets 拉资产列表(分页,单页)
//
// 端点:GET {baseURL}/api/v1/assets/assets/?limit=N&offset=M
// 第一次调 offset=0,然后取 resp.Next URL 继续(里面自带 offset/limit)
//
// 返回的 AssetListResp.Next == nil 表示全部拉完
func (c *Client) ListAssets(url string) (*AssetListResp, error) {
	if url == "" {
		// 第一次调用
		url = c.baseURL + "/api/v1/assets/assets/?limit=100"
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("X-JMS-ORG", c.orgID)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("JMS 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("JMS GET assets -> %d: %s", resp.StatusCode, string(body))
	}

	var out AssetListResp
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("JMS 响应解析失败: %w (body 前 200 字符: %.200s)", err, string(body))
	}
	return &out, nil
}
