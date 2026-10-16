package model

import (
	"time"

	"go_server/config"
)

// OperationLog 操作日志
//
// 2026-10-16 新增:统一记 4 类业务事件,用于审计 + 复盘
//   - deploy      POST /api/codeDeploy/packages/:id/pull
//   - upload      POST /api/codeDeploy/packages
//   - delete      DELETE /api/codeDeploy/packages/:id
//   - download    GET  /api/codeDeploy/packages/:id/download-url
//   - project.sync POST /api/codeDeploy/projects/sync
//   - asset.sync  POST /api/codeDeploy/assets/sync
//   - sync.all    POST /api/codeDeploy/sync
//
// 设计原则:
//   - Target* / Username 冗余存,即便包/项目/资产被删,日志仍能看历史
//   - 不软删(BaseModel 有软删字段,日志保留全量更稳)
//   - JMS 远端输出末 2000 字符存 JMSOutput,失败时直接展示给前端看
type OperationLog struct {
	ID uint `gorm:"primarykey" json:"id"`

	// 事件
	Event  string `gorm:"size:32;index;not null" json:"event"`    // 事件类型
	Result int    `gorm:"default:1;index" json:"result"`          // 1=成功 0=失败
	Method string `gorm:"size:10" json:"method"`                  // HTTP method
	Path   string `gorm:"size:255;index" json:"path"`             // 接口 path

	// 操作人
	UserID   uint   `gorm:"index" json:"user_id"`
	Username string `gorm:"size:64;index" json:"username"` // 冗余

	// 目标(冗余,即便被删仍能看历史)
	TargetType string `gorm:"size:32;index" json:"target_type"` // "package" / "project" / "asset"
	TargetID   uint   `gorm:"index" json:"target_id"`
	TargetName string `gorm:"size:200" json:"target_name"`       // 冗余展示名

	// 时间
	StartedAt  time.Time  `gorm:"index" json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	DurationMS int64       `json:"duration_ms"`

	// 部署专项(Event=deploy 时填)
	JMSTaskID   string `gorm:"size:64" json:"jms_task_id"`
	JMSAssetName string `gorm:"size:200" json:"jms_asset_name"`
	JMSOutput   string `gorm:"type:text" json:"jms_output"` // stdout/stderr 末 2000 字符
	ErrorMsg    string `gorm:"size:500" json:"error_msg"`

	// HTTP 上下文
	ClientIP string `gorm:"size:64" json:"client_ip"`
	UserAgent string `gorm:"size:200" json:"user_agent"`

	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (OperationLog) TableName() string {
	return config.AppConfig.Database.Prefix + "operation_log"
}
