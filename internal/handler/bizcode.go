// Package handler 提供 HTTP 业务码常量
//
// 业务码规范(与前端 src/constants/bizcode.js 保持一致):
//  0       成功
//  1xxx    认证 / 账号 / 角色
//  2xxx    权限
//  3xxx    文件 / OSS
//  4xxx    参数
//  9xxx    通用 / 未分类
//
// 加新业务码的流程:
//  1. 在这里选合适段位加常量
//  2. 在 service 里定义 ErrXxx = errors.New(...)
//  3. handler 用 errors.Is 翻译 + handler.Error(c, CodeXxx, msg)
//  4. 前端 src/constants/bizcode.js 同步加常量
//
// 后端 HTTP 状态码**恒为 200**,真实状态在 body.code 里(国内后台项目标准做法)。
package handler

// 业务码定义
const (
	// CodeSuccess 成功
	CodeSuccess = 0

	// 1xxx 认证 / 账号 / 角色
	CodeAuthFail      = 1001 // 令牌生成失败
	CodeTokenInvalid  = 1002 // 令牌无效或已过期
	CodeTokenMissing  = 1003 // 未携带令牌
	CodeUserNotFound  = 1004 // 用户不存在
	CodeUserDuplicate = 1005 // 用户名已存在
	CodeUserPassword  = 1006 // 密码错误
	CodeUserNoRole    = 1007 // 该用户未分配角色
	CodeRoleNotFound  = 1008 // 角色不存在
	CodeRoleDuplicate = 1009 // 角色名称已存在
	CodeMenuNotFound  = 1010 // 菜单不存在

	// 2xxx 权限
	CodeNoPermission = 2001 // 无权限

	// 3xxx 文件 / OSS
	CodeUploadInvalid = 3001 // 请选择要上传的文件
	CodeUploadType    = 3002 // 不支持的图片格式
	CodeUploadSize    = 3003 // 图片大小超过限制
	CodeOSSNoConfig   = 3004 // OSS 未配置
	CodeOSSClient     = 3005 // OSS 客户端创建失败
	CodeOSSBucket     = 3006 // OSS Bucket 获取失败
	CodeUploadRead    = 3007 // 文件读取失败
	CodeUploadPut     = 3008 // 文件上传失败

	// 4xxx 参数
	CodeParamsInvalid = 4001 // 参数错误(通用)

	// 9xxx 通用
	CodeUnknown = 9999 // 未分类失败
)
