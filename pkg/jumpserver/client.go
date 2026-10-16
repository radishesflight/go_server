// Package jumpserver 提供精简版 JumpServer Core API 客户端
//
// 设计原则:
//   - 只覆盖本项目实际用到的端点,不全量实现
//   - 凭据(base_url / token / org_id)由调用方从 config.AppConfig.Jumpserver 传入
//     本 package 不读任何全局配置,保持纯函数式
//   - 返回值尽量用领域类型(Node / Asset)而不是裸 map
//
// 当前实现的端点:
//   - GET  /api/v1/assets/nodes/children/tree/?assets=0&     拉资产根节点树
//   - GET  /api/v1/assets/assets/?limit=N&offset=M            拉资产列表(分页)
//   - POST /api/v1/ops/jobs/                                  提交 ad-hoc 任务
//   - GET  /api/v1/ops/job-execution/task-detail/<task_id>/   轮询任务状态
//   - GET  /api/v1/ops/job-executions/?job_id=<job_id>        拿执行结果(stdout/material/summary)
//   - DELETE /api/v1/ops/jobs/<job_id>/                       清理任务
//
// 不做什么:
//   - 不实现 Koko SSH 通道(那是 web 终端专用,这里只跑命令用 Core API)
//   - 不做连接池(每次 NewClient 一个 http.Client,30s timeout;RunCommand 内部用更长 timeout 包装)
package jumpserver

import (
	"bytes"
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

// ========================================================================
// ad-hoc 命令执行:用于"代码包拉取"场景(对单台资产跑 cd+curl+unzip 串)
//
// 流程:
//  1. CreateJob     POST   /api/v1/ops/jobs/              → 拿到 task_id
//  2. PollTask      GET    /api/v1/ops/job-execution/task-detail/<task_id>/
//  3. GetJobExecutionDetail  GET /api/v1/ops/job-executions/?job_id=<job_id>
//  4. DeleteJob     DELETE /api/v1/ops/jobs/<job_id>/
//
// RunCommand 是顶层包装,内部完成 1→2→3→4 全部,失败带 context。
// ========================================================================

// CreateJob 提交一条 ad-hoc 任务,对单台资产跑一条 shell 命令
//
//	端点:POST /api/v1/ops/jobs/
//	参数:assetID=JMS 资产 uuid,command=shell 命令字符串
//	返回:task_id(给 PollTask 用)
func (c *Client) CreateJob(assetID, command string) (string, error) {
	payload := map[string]interface{}{
		"name":         fmt.Sprintf("codedeploy-%d", time.Now().Unix()),
		"type":         "adhoc",
		"module":       "shell",
		"args":         command,
		"assets":       []string{assetID},
		"runas_policy": "skip",
		"runas":        "root",
		"instant":      true,
		"crontab":      "0 0 * * *",
		"comment":      "from go_server codeDeploy",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("构造请求体失败: %w", err)
	}
	url := c.baseURL + "/api/v1/ops/jobs/"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("X-JMS-ORG", c.orgID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("JMS CreateJob 请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("JMS POST /ops/jobs/ -> %d: %s", resp.StatusCode, string(respBody))
	}
	var r struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(respBody, &r); err != nil {
		return "", fmt.Errorf("JMS CreateJob 响应解析失败: %w (body: %.200s)", err, string(respBody))
	}
	if r.TaskID == "" {
		return "", fmt.Errorf("JMS CreateJob 响应里没有 task_id (body: %.200s)", string(respBody))
	}
	return r.TaskID, nil
}

// TaskDetail 任务执行详情(轮询接口返回)
type TaskDetail struct {
	IsFinished bool    `json:"is_finished"`
	IsSuccess  bool    `json:"is_success"`
	Status     string  `json:"status"` // running / success / failed / ...
	JobID      string  `json:"job_id"`
	TimeCost   float64 `json:"time_cost"`
}

// PollTask 轮询任务直到完成
//
//	端点:GET /api/v1/ops/job-execution/task-detail/<task_id>/
//	轮询间隔 2s,timeout 由调用方控制(典型 5min 容纳 100MB 包下载+解压)
func (c *Client) PollTask(taskID string, timeout time.Duration) (*TaskDetail, error) {
	deadline := time.Now().Add(timeout)
	url := c.baseURL + "/api/v1/ops/job-execution/task-detail/" + taskID + "/"
	for {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("构造请求失败: %w", err)
		}
		req.Header.Set("Authorization", "Token "+c.token)
		req.Header.Set("X-JMS-ORG", c.orgID)
		req.Header.Set("Accept", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("JMS PollTask 请求失败: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("JMS GET task-detail -> %d: %s", resp.StatusCode, string(body))
		}
		var td TaskDetail
		if err := json.Unmarshal(body, &td); err != nil {
			return nil, fmt.Errorf("JMS task-detail 解析失败: %w (body: %.200s)", err, string(body))
		}
		if td.IsFinished {
			return &td, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("JMS 任务轮询超时(最后状态: %s,已等 %v)", td.Status, timeout)
		}
		time.Sleep(2 * time.Second)
	}
}

// GetJobExecutionDetail 拿执行结果(含 material=stdout 和 summary=exit_code/duration 等)
//
//	端点:GET /api/v1/ops/job-executions/?job_id=<job_id>
func (c *Client) GetJobExecutionDetail(jobID string) (map[string]interface{}, error) {
	url := c.baseURL + "/api/v1/ops/job-executions/?job_id=" + jobID
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("X-JMS-ORG", c.orgID)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("JMS GetJobExecutionDetail 请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("JMS GET job-executions -> %d: %s", resp.StatusCode, string(body))
	}
	// 响应可能直接是数组,也可能分页包了 results,两种都吃
	var arr []map[string]interface{}
	if err := json.Unmarshal(body, &arr); err == nil && len(arr) > 0 {
		return arr[0], nil
	}
	var wrapped struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && len(wrapped.Results) > 0 {
		return wrapped.Results[0], nil
	}
	return nil, fmt.Errorf("JMS 找不到 job_id=%s 的 JobExecution (body: %.200s)", jobID, string(body))
}

// DeleteJob 清理任务
//
//	端点:DELETE /api/v1/ops/jobs/<job_id>/
//	失败仅 log,不影响主流程(任务历史会自然过期)
func (c *Client) DeleteJob(jobID string) error {
	url := c.baseURL + "/api/v1/ops/jobs/" + jobID + "/"
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Token "+c.token)
	req.Header.Set("X-JMS-ORG", c.orgID)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("JMS DeleteJob 请求失败: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("JMS DELETE job -> %d", resp.StatusCode)
	}
	return nil
}

// RunCommand 顶层包装:对单台资产同步跑一条命令,等终态,拿 stdout + summary
//
//	assetID: JMS 资产 uuid
//	command: shell 命令字符串
//	timeout: 总超时(含 100MB 包下载/解压,默认 5min)
//
// 返回:map 字段:
//   - is_finished / is_success / status / time_cost
//   - material     资产上的 stdout(含 echo 标记的部署结果)
//   - summary      map{exit_code?, duration?, ...}(如 JMS 提供)
//
// 错误:任一步骤失败返回 err;若命令本身执行失败(exit_code != 0 或 status=failed)返回 result+err
//
//	方便上层把 stdout/material 透回给前端看。
func (c *Client) RunCommand(assetID, command string, timeout time.Duration) (map[string]interface{}, error) {
	taskID, err := c.CreateJob(assetID, command)
	if err != nil {
		return nil, fmt.Errorf("创建 JMS 任务失败: %w", err)
	}
	td, err := c.PollTask(taskID, timeout)
	if err != nil {
		return nil, fmt.Errorf("轮询 JMS 任务失败: %w", err)
	}
	result := map[string]interface{}{
		"is_finished": td.IsFinished,
		"is_success":  td.IsSuccess,
		"status":      td.Status,
		"time_cost":   td.TimeCost,
		"task_id":     taskID,
	}
	if td.JobID != "" {
		if je, jerr := c.GetJobExecutionDetail(td.JobID); jerr == nil {
			if m, ok := je["material"].(string); ok {
				result["material"] = m
			}
			if sm, ok := je["summary"].(map[string]interface{}); ok {
				result["summary"] = sm
			}
		}
		// 任务执行完即可清理,失败仅 log
		if derr := c.DeleteJob(td.JobID); derr != nil {
			// 静默,不污染返回
			_ = derr
		}
	}
	if !td.IsSuccess {
		return result, fmt.Errorf("JMS 命令执行失败 (status=%s)", td.Status)
	}
	return result, nil
}
