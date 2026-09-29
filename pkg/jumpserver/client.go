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

// NewClient 构造一个 Client
//   - baseURL: e.g. "http://118.24.89.187:8081"
//   - token:   JMS 用户的 Token(从 JMS 个人中心拿)
//   - orgID:   组织 ID(多组织场景用,UUID 字符串)
func NewClient(baseURL, token, orgID string) *Client {
	return &Client{
		baseURL: baseURL,
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
