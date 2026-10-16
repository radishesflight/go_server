// Package service asset_deps_probe.go - 资产部署依赖探测
//
// 用途:在部署代码包前,探测目标 JMS 资产上是否装了 zip / unzip / curl 这些
// 部署命令。如果缺,部署会 rc=127(command not found)失败,但前端只能看到
// 笼统的"部署失败",看不到具体哪个命令缺。提前探测能给出明确反馈。
//
// 实现细节(必须理解清楚):
//   JMS ad-hoc 任务执行后,只有"成功/失败"的概要,真实 stdout **只**在任务失败时
//   通过 summary.failures[<asset>] 字段返回(任务成功时,stdout 根本不返回)。
//   所以我们让探测脚本故意 `exit 1`,把 stdout 通过 failures 带回来。
//   业务上"探测完成"对应 JMS 任务"失败",service.ProbeAssetDeps 不看 is_success,
//   只看 is_finished,然后解析 failures 拿到 stdout。
//
// 调用链:
//   handler.ProbeAssetDeps(c) -> service.ProbeAssetDeps(assetID)
//     -> jumpserver.Client.CreateJob()  (探测脚本)
//     -> jumpserver.Client.PollTask()   (等终态)
//     -> jumpserver.Client.GetJobExecutionDetail() (拿 summary.failures)
//     -> parse tool list
package service

import (
	"fmt"
	"strings"
	"time"

	"go_server/config"
	"go_server/internal/model"
	"go_server/pkg/jumpserver"
)

// AssetDepsResult 资产依赖探测结果
type AssetDepsResult struct {
	AssetID   uint                `json:"asset_id"`
	AssetName string              `json:"asset_name"`
	AllOK     bool                `json:"all_ok"`     // zip/unzip/curl 全部就绪
	Tools     map[string]ToolInfo `json:"tools"`      // key: zip/unzip/curl
}

// ToolInfo 单个工具的探测结果
type ToolInfo struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path"`              // which 出来的绝对路径
	Missing bool   `json:"missing,omitempty"` // 缺工具时 = true
}

// ProbeAssetDeps 在指定 JMS 资产上跑探测命令,返回 zip/unzip/curl 三个工具的可用状态
func (s *BusinessAssetService) ProbeAssetDeps(assetID uint) (*AssetDepsResult, error) {
	if assetID == 0 {
		return nil, ErrAssetNotFound
	}
	var asset model.BusinessAssets
	if err := model.DB.First(&asset, assetID).Error; err != nil {
		return nil, ErrAssetNotFound
	}
	if asset.Status != 1 {
		return nil, ErrAssetNotFound
	}
	if asset.JmsID == "" {
		return nil, ErrAssetNotFound
	}

	// 探测脚本:对每个工具 echo "${tool}=$(which $tool 2>/dev/null || echo MISSING)"
	//         末尾 echo __PROBE_END__ 当作后端解析的终止符
	//         最后 exit 1 — 故意失败让 JMS 把 stdout 抓进 summary.failures
	//
	// 关键:所有 echo 重定向到 stderr(1>&2)。JMS ad-hoc 任务的 summary.failures
	//     字段只抓 stderr 不抓 stdout(只 echo 写 stdout 的话,summary 是空的,
	//     只剩 "shell: non-zero return code")。这点跟部署脚本的 unzip/tail 报错
	//     一致(它们都是写 stderr 才被 JMS 抓到)。
	probeCmd := `for tool in zip unzip curl; do echo "${tool}=$(which $tool 2>/dev/null || echo MISSING)" 1>&2; done; echo "__PROBE_END__" 1>&2; exit 1`

	jmsClient := jumpserver.NewClient(
		config.AppConfig.Jumpserver.BaseURL,
		config.AppConfig.Jumpserver.Token,
		config.AppConfig.Jumpserver.OrgID,
	)

	taskID, err := jmsClient.CreateJob(asset.JmsID, probeCmd)
	if err != nil {
		return nil, fmt.Errorf("创建探测任务失败: %w", err)
	}
	td, err := jmsClient.PollTask(taskID, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("轮询探测任务失败: %w", err)
	}

	// 从 job-execution 拿 stdout(在 summary.failures[<asset>] 里,故意失败才有)
	var rawOutput string
	if td.JobID != "" {
		if je, jerr := jmsClient.GetJobExecutionDetail(td.JobID); jerr == nil {
			if sm, ok := je["summary"].(map[string]interface{}); ok {
				if fails, ok := sm["failures"].(map[string]interface{}); ok {
					// 优先用 asset.Name 精确匹配,fallback 到第一个 failure(单资产任务就一条)
					if v, ok := fails[asset.Name].(string); ok {
						rawOutput = v
					} else {
						for _, v := range fails {
							if s, ok := v.(string); ok {
								rawOutput = s
								break
							}
						}
					}
				}
			}
		}
	}

	// DEBUG: 输出 rawOutput 真实样子
	_ = rawOutput // 不再 debug,保留获取步骤

	// 解析 output:每行 "tool=path" 或 "tool=MISSING",遇到 __PROBE_END__ 停
	tools := map[string]ToolInfo{
		"zip":   {OK: false, Missing: true},
		"unzip": {OK: false, Missing: true},
		"curl":  {OK: false, Missing: true},
	}
	for _, line := range strings.Split(rawOutput, "\n") {
		line = strings.TrimSpace(line)
		// 剥 JMS 加的 "shell: " 前缀 和 "; non-zero return code" 后缀
		if strings.HasPrefix(line, "shell:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "shell:"))
		}
		if idx := strings.Index(line, ";non-zero"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if name == "__PROBE_END__" {
			break
		}
		info, ok := tools[name]
		if !ok {
			continue
		}
		if val == "MISSING" || val == "" {
			info.OK = false
			info.Missing = true
		} else {
			info.OK = true
			info.Path = val
			info.Missing = false
		}
		tools[name] = info
	}
	// 任何工具不 OK → allOK = false(注意:即使循环没匹配到任何 tool,3 个 tools 仍
	// 是初始 {OK: false, Missing: true},allOK 也会 = false — 这正好是预期)
	allOK := true
	for _, info := range tools {
		if !info.OK {
			allOK = false
			break
		}
	}

	return &AssetDepsResult{
		AssetID:   asset.ID,
		AssetName: asset.Name,
		AllOK:     allOK,
		Tools:     tools,
	}, nil
}
