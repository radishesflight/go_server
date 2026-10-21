// Package system admin_users.go - 管理员(用户)管理 HTTP 入口
//
// 接口列表:
//
//	GET    /api/system/adminUsers/list  列表分页查询
//	GET    /api/system/adminUsers/:id   单条查询
//	POST   /api/system/adminUsers       新增
//	PUT    /api/system/adminUsers/:id   更新
//	DELETE /api/system/adminUsers/:id   删除(软删除)
//
// 业务码翻译表(与 internal/handler/bizcode.go 一致):
//
//	service.ErrUserNotFound      → CodeUserNotFound   (1004)
//	service.ErrUserNameDuplicate → CodeUserDuplicate  (1005)
//	service.ErrUserPasswordHash  → CodeUnknown        (9999,内部错误)
//
// 角色下拉数据来源:调 getAdminRolesList 接口(/api/system/adminRoles/list)
package system

import (
	"errors"
	"strconv"

	"go_server/internal/handler"
	"go_server/internal/service"

	"github.com/gin-gonic/gin"
)

// adminUserSvc 用户管理业务入口
var adminUserSvc = service.NewAdminUserService()

// AdminUsersQuery 列表查询参数(query string)
type AdminUsersQuery struct {
	Page   int `form:"page" json:"page"`     // 页码(从 1 开始,默认 1)
	Size   int `form:"size" json:"size"`     // 每页条数(默认 10)
	Status int `form:"status" json:"status"` // 状态筛选(0=全部,1=启用,2=禁用)
}

// CreateAdminUsersReq 创建请求体
type CreateAdminUsersReq struct {
	Username string `json:"username" binding:"required"` // 用户名(必填,唯一)
	Password string `json:"password" binding:"required"` // 密码(必填,会被 bcrypt 加密)
	Email    string `json:"email"`                       // 邮箱(可选)
	Phone    string `json:"phone"`                       // 手机号(可选)
	Status   int    `json:"status"`                      // 状态(1=启用,0=禁用)
	RoleID   uint   `json:"role_id"`                     // 角色 ID(可选,0=未分配)
}

// UpdateAdminUsersReq 更新请求体
type UpdateAdminUsersReq struct {
	Email  string `json:"email"`
	Phone  string `json:"phone"`
	Status int    `json:"status"`
	RoleID uint   `json:"role_id"`
}

// GetAdminUsersList 分页查询用户列表
func GetAdminUsersList(c *gin.Context) {
	var query AdminUsersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	list, total, _ := adminUserSvc.GetList(query.Page, query.Size, query.Status)

	handler.Success(c, gin.H{
		"list":  list,
		"total": total,
		"page":  query.Page,
		"size":  query.Size,
	})
}

// GetAdminUsers 单条用户
func GetAdminUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		return
	}

	u, err := adminUserSvc.Get(id)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			handler.Error(c, handler.CodeUserNotFound, "用户不存在")
		} else {
			handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		}
		return
	}
	handler.Success(c, u)
}

// CreateAdminUsers 创建用户
func CreateAdminUsers(c *gin.Context) {
	var req CreateAdminUsersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	u, err := adminUserSvc.Create(req.Username, req.Password, req.Email, req.Phone, req.Status, req.RoleID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNameDuplicate):
			handler.Error(c, handler.CodeUserDuplicate, "用户名已存在")
		case errors.Is(err, service.ErrUserPasswordHash):
			handler.Error(c, handler.CodeUnknown, "密码加密失败")
		default:
			handler.Error(c, handler.CodeUnknown, "创建用户失败")
		}
		return
	}
	handler.Success(c, u)
}

// UpdateAdminUsers 更新用户
func UpdateAdminUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		return
	}

	var req UpdateAdminUsersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	u, err := adminUserSvc.Update(id, req.Email, req.Phone, req.Status, req.RoleID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			handler.Error(c, handler.CodeUserNotFound, "用户不存在")
		} else {
			handler.Error(c, handler.CodeUnknown, "更新用户失败")
		}
		return
	}
	handler.Success(c, u)
}

// DeleteAdminUsers 删除用户
func DeleteAdminUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		return
	}

	if err := adminUserSvc.Delete(id); err != nil {
		handler.Error(c, handler.CodeUnknown, "删除用户失败")
		return
	}
	handler.Success(c, nil)
}

// UnlockAdminUser 解除登录限制(2026-10-20 新增)
//
// POST /api/system/adminUsers/unlock/:id
// 业务码:
//
//	service.ErrUserInvalidID → CodeParamsInvalid
//	service.ErrUserNotFound  → CodeUserNotFound
//	service.ErrUserUnlock    → CodeUnknown
//
// 权限:由 PermissionMiddleware 控(必须 admin 角色拥有该 op)
// 行为:清零 failed_count + locked_at,允许用户重新登录
func UnlockAdminUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		return
	}

	if err := adminUserSvc.UnlockUser(id); err != nil {
		switch {
		case errors.Is(err, service.ErrUserInvalidID):
			handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		case errors.Is(err, service.ErrUserNotFound):
			handler.Error(c, handler.CodeUserNotFound, "用户不存在")
		default:
			handler.Error(c, handler.CodeUnknown, "解除登录限制失败")
		}
		return
	}
	handler.Success(c, nil)
}

// UpdateMyProfileReq 个人中心 - 当前用户改自己的请求体
//
// 2026-10-19 新增
//   - email / phone / newPassword: 都可选,空字符串 = 不改
//   - oldPassword: 必填(即使不改密码也要输,做身份二次确认)
type UpdateMyProfileReq struct {
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	NewPassword string `json:"new_password"`
	OldPassword string `json:"old_password" binding:"required"`
}

// UpdateMyProfile 个人中心 - 当前用户改自己
//
// 2026-10-19 新增:挂在 AuthMiddleware 下,不走 PermissionMiddleware,
//
//	所有登录用户(包括普通角色如"前端")都能调
//	id 从 c.Get("user_id") 拿,前端不传,防越权改他人
//
// PUT /api/system/adminUsers/me
func UpdateMyProfile(c *gin.Context) {
	var req UpdateMyProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		handler.Error(c, handler.CodeParamsInvalid, "参数错误")
		return
	}

	// 从 token 拿当前用户 id(AuthMiddleware 已注入)
	uidVal, exists := c.Get("user_id")
	if !exists {
		handler.Error(c, handler.CodeTokenMissing, "未登录")
		return
	}
	uid, ok := uidVal.(uint)
	if !ok {
		// AuthMiddleware 注入的可能是 uint64,兼容一下
		switch v := uidVal.(type) {
		case uint64:
			uid = uint(v)
		default:
			handler.Error(c, handler.CodeUnknown, "用户ID类型异常")
			return
		}
	}

	u, err := adminUserSvc.UpdateMe(uid, req.Email, req.Phone, req.NewPassword, req.OldPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			handler.Error(c, handler.CodeUserNotFound, "用户不存在")
		case errors.Is(err, service.ErrUserPasswordWrong):
			handler.Error(c, handler.CodeUserPassword, "当前密码错误")
		case errors.Is(err, service.ErrUserPasswordHash):
			handler.Error(c, handler.CodeUnknown, "密码加密失败")
		case errors.Is(err, service.ErrUserInvalidID):
			handler.Error(c, handler.CodeParamsInvalid, "无效的用户ID")
		default:
			handler.Error(c, handler.CodeUnknown, "更新失败")
		}
		return
	}
	handler.Success(c, u)
}
