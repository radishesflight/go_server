package handler

// 业务码定义
//  0       成功
//  1xxx    认证 / 账号 / 角色
//  2xxx    权限
//  3xxx    文件 / OSS
//  4xxx    参数
//  9xxx    通用 / 未分类
const (
	CodeSuccess = 0

	// 1xxx 认证 / 账号 / 角色
	CodeAuthFail      = 1001
	CodeTokenInvalid  = 1002
	CodeTokenMissing  = 1003
	CodeUserNotFound  = 1004
	CodeUserDuplicate = 1005
	CodeUserPassword  = 1006
	CodeUserNoRole    = 1007
	CodeRoleNotFound  = 1008
	CodeRoleDuplicate = 1009
	CodeMenuNotFound  = 1010

	// 2xxx 权限
	CodeNoPermission = 2001

	// 3xxx 文件 / OSS
	CodeUploadInvalid = 3001
	CodeUploadType    = 3002
	CodeUploadSize    = 3003
	CodeOSSNoConfig   = 3004
	CodeOSSClient     = 3005
	CodeOSSBucket     = 3006
	CodeUploadRead    = 3007
	CodeUploadPut     = 3008

	// 4xxx 参数
	CodeParamsInvalid = 4001

	// 9xxx 通用
	CodeUnknown = 9999
)
