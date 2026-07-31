// Package main seed 工具 - 初始化基础数据
//
// 跑法:go run cmd/seed/main.go
// 会:
//  1. 清空所有业务表
//  2. 插入种子数据:部门 / 角色 / 菜单 / operations / 用户 / 角色-菜单 / 角色-操作
//
// 种子账号:
//   admin / 123456(超级管理员,所有权限)
//   test2 / 123456(测试员,只有"角色权限分配"菜单)
//
// 注意:会**清空所有表**,只用于初始化/重置环境
package main

import (
	"fmt"
	"log"
	"time"

	"go_server/config"
	"go_server/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 1. 读配置 + 初始化 DB
	if err := config.InitConfig("config/config.yaml"); err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		config.AppConfig.Database.User,
		config.AppConfig.Database.Password,
		config.AppConfig.Database.Host,
		config.AppConfig.Database.Port,
		config.AppConfig.Database.Dbname,
		config.AppConfig.Database.Charset,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	model.DB = db
	now := time.Now()

	// 2. 清空所有业务表(按外键依赖倒序)
	// 用 Unscoped() 走硬删除,默认 Delete 是软删除(deleted_at),行还在
	tables := []interface{}{
		&model.AdminRoleOperations{},
		&model.AdminRoleMenus{},
		&model.AdminMenuOperations{},
		&model.AdminUsers{},
		&model.AdminRoles{},
		&model.AdminMenus{},
		&model.AdminDepartments{},
	}
	for _, t := range tables {
		if err := db.Unscoped().Where("1 = 1").Delete(t).Error; err != nil {
			log.Printf("清空表 %T 失败(可能表不存在,跳过): %v", t, err)
		}
	}
	fmt.Println("✓ 清空旧数据(硬删除)")

	// 3. 部门
	depts := []model.AdminDepartments{
		{BaseModel: model.BaseModel{ID: 1, CreatedAt: now, UpdatedAt: now}, Name: "总公司", ParentID: 0, Sort: 0, Status: 1},
		{BaseModel: model.BaseModel{ID: 2, CreatedAt: now, UpdatedAt: now}, Name: "研发部", ParentID: 1, Sort: 1, Status: 1},
		{BaseModel: model.BaseModel{ID: 3, CreatedAt: now, UpdatedAt: now}, Name: "测试部", ParentID: 1, Sort: 2, Status: 1},
	}
	if err := db.Create(&depts).Error; err != nil {
		log.Fatalf("插入部门失败: %v", err)
	}
	fmt.Println("✓ 部门:总公司 / 研发部 / 测试部")

	// 4. 角色
	roles := []model.AdminRoles{
		{BaseModel: model.BaseModel{ID: 1, CreatedAt: now, UpdatedAt: now}, Name: "超级管理员", Describe: "拥有所有权限", Status: 1, DataScope: 0},
		{BaseModel: model.BaseModel{ID: 2, CreatedAt: now, UpdatedAt: now}, Name: "测试员", Describe: "测试用角色", Status: 1, DataScope: 0},
	}
	if err := db.Create(&roles).Error; err != nil {
		log.Fatalf("插入角色失败: %v", err)
	}
	fmt.Println("✓ 角色:超级管理员 / 测试员")

	// 5. 菜单
	menus := []model.AdminMenus{
		{BaseModel: model.BaseModel{ID: 1, CreatedAt: now, UpdatedAt: now}, Name: "首页", Code: "dashboard", Path: "/dashboard", Icon: "HomeFilled", ParentID: 0, Sort: 100, Status: 1, DataScope: 0},
		{BaseModel: model.BaseModel{ID: 2, CreatedAt: now, UpdatedAt: now}, Name: "系统管理", Code: "system", Path: "/system", Icon: "Setting", ParentID: 0, Sort: 90, Status: 1, DataScope: 0},
		{BaseModel: model.BaseModel{ID: 3, CreatedAt: now, UpdatedAt: now}, Name: "管理员管理", Code: "adminUsers", Path: "/system/adminUsers", Icon: "User", ParentID: 2, Sort: 91, Status: 1, DataScope: 0},
		{BaseModel: model.BaseModel{ID: 4, CreatedAt: now, UpdatedAt: now}, Name: "权限分配", Code: "roleMenu", Path: "/system/roleMenu", Icon: "Key", ParentID: 2, Sort: 92, Status: 1, DataScope: 0},
		{BaseModel: model.BaseModel{ID: 5, CreatedAt: now, UpdatedAt: now}, Name: "菜单管理", Code: "adminMenus", Path: "/system/adminMenus", Icon: "Menu", ParentID: 2, Sort: 93, Status: 1, DataScope: 0},
		{BaseModel: model.BaseModel{ID: 6, CreatedAt: now, UpdatedAt: now}, Name: "部门管理", Code: "departments", Path: "/system/departments", Icon: "OfficeBuilding", ParentID: 2, Sort: 94, Status: 1, DataScope: 0},
		{BaseModel: model.BaseModel{ID: 7, CreatedAt: now, UpdatedAt: now}, Name: "角色管理", Code: "adminRoles", Path: "/system/adminRoles", Icon: "Avatar", ParentID: 2, Sort: 95, Status: 1, DataScope: 0},
	}
	if err := db.Create(&menus).Error; err != nil {
		log.Fatalf("插入菜单失败: %v", err)
	}
	fmt.Println("✓ 菜单:6 个(首页 / 系统管理 / 4 个子菜单)")

	// 6. operations
	ops := []model.AdminMenuOperations{
		// 管理员管理 (menu_id=3)
		{BaseModel: model.BaseModel{ID: 1, CreatedAt: now, UpdatedAt: now}, MenuID: 3, Code: "view", Name: "查看", Sort: 1},
		{BaseModel: model.BaseModel{ID: 2, CreatedAt: now, UpdatedAt: now}, MenuID: 3, Code: "add", Name: "新增", Sort: 2},
		{BaseModel: model.BaseModel{ID: 3, CreatedAt: now, UpdatedAt: now}, MenuID: 3, Code: "edit", Name: "编辑", Sort: 3},
		{BaseModel: model.BaseModel{ID: 4, CreatedAt: now, UpdatedAt: now}, MenuID: 3, Code: "delete", Name: "删除", Sort: 4},
		// 权限分配 (menu_id=4)
		{BaseModel: model.BaseModel{ID: 5, CreatedAt: now, UpdatedAt: now}, MenuID: 4, Code: "view", Name: "查看", Sort: 1},
		{BaseModel: model.BaseModel{ID: 6, CreatedAt: now, UpdatedAt: now}, MenuID: 4, Code: "edit", Name: "编辑", Sort: 3},
		// 菜单管理 (menu_id=5)
		{BaseModel: model.BaseModel{ID: 7, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Code: "view", Name: "查看", Sort: 1},
		{BaseModel: model.BaseModel{ID: 8, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Code: "add", Name: "新增", Sort: 2},
		{BaseModel: model.BaseModel{ID: 9, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Code: "edit", Name: "编辑", Sort: 3},
		{BaseModel: model.BaseModel{ID: 10, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Code: "delete", Name: "删除", Sort: 4},
		// 部门管理 (menu_id=6)
		{BaseModel: model.BaseModel{ID: 11, CreatedAt: now, UpdatedAt: now}, MenuID: 6, Code: "view", Name: "查看", Sort: 1},
		{BaseModel: model.BaseModel{ID: 12, CreatedAt: now, UpdatedAt: now}, MenuID: 6, Code: "add", Name: "新增", Sort: 2},
		{BaseModel: model.BaseModel{ID: 13, CreatedAt: now, UpdatedAt: now}, MenuID: 6, Code: "edit", Name: "编辑", Sort: 3},
		{BaseModel: model.BaseModel{ID: 14, CreatedAt: now, UpdatedAt: now}, MenuID: 6, Code: "delete", Name: "删除", Sort: 4},
		// 角色管理 (menu_id=7)
		{BaseModel: model.BaseModel{ID: 15, CreatedAt: now, UpdatedAt: now}, MenuID: 7, Code: "view", Name: "查看", Sort: 1},
		{BaseModel: model.BaseModel{ID: 16, CreatedAt: now, UpdatedAt: now}, MenuID: 7, Code: "add", Name: "新增", Sort: 2},
		{BaseModel: model.BaseModel{ID: 17, CreatedAt: now, UpdatedAt: now}, MenuID: 7, Code: "edit", Name: "编辑", Sort: 3},
		{BaseModel: model.BaseModel{ID: 18, CreatedAt: now, UpdatedAt: now}, MenuID: 7, Code: "delete", Name: "删除", Sort: 4},
	}
	if err := db.Create(&ops).Error; err != nil {
		log.Fatalf("插入 operations 失败: %v", err)
	}
	fmt.Println("✓ operations:14 条(每个菜单 2-4 个)")

	// 7. 用户(密码 123456,bcrypt 加密)
	hash := func(p string) string {
		h, _ := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
		return string(h)
	}
	users := []model.AdminUsers{
		{BaseModel: model.BaseModel{ID: 1, CreatedAt: now, UpdatedAt: now}, Username: "admin", Password: hash("123456"), Email: "admin@example.com", Status: 1, RoleID: 1, DepartmentID: 1},
		{BaseModel: model.BaseModel{ID: 2, CreatedAt: now, UpdatedAt: now}, Username: "test2", Password: hash("123456"), Email: "test2@example.com", Status: 1, RoleID: 2, DepartmentID: 3},
	}
	if err := db.Create(&users).Error; err != nil {
		log.Fatalf("插入用户失败: %v", err)
	}
	fmt.Println("✓ 用户:admin / test2(密码都是 123456)")

	// 8. 角色-菜单(超级管理员:全部,测试员:首页 + 系统管理 + 角色权限分配)
	roleMenus := []model.AdminRoleMenus{
		// 超级管理员
		{RoleID: 1, MenuID: 1}, {RoleID: 1, MenuID: 2}, {RoleID: 1, MenuID: 3},
		{RoleID: 1, MenuID: 4}, {RoleID: 1, MenuID: 5}, {RoleID: 1, MenuID: 6},
		{RoleID: 1, MenuID: 7},
		// 测试员(也要看角色列表 + 编辑 + 分配权限)
		{RoleID: 2, MenuID: 1}, {RoleID: 2, MenuID: 2}, {RoleID: 2, MenuID: 4},
		{RoleID: 2, MenuID: 7},
	}
	if err := db.Create(&roleMenus).Error; err != nil {
		log.Fatalf("插入角色-菜单失败: %v", err)
	}
	fmt.Println("✓ 角色-菜单:超级管理员 6 个 / 测试员 3 个")

	// 9. 角色-操作
	roleOps := []model.AdminRoleOperations{
		// 超级管理员:所有操作
		{RoleID: 1, MenuID: 3, OperationID: 1}, {RoleID: 1, MenuID: 3, OperationID: 2},
		{RoleID: 1, MenuID: 3, OperationID: 3}, {RoleID: 1, MenuID: 3, OperationID: 4},
		{RoleID: 1, MenuID: 4, OperationID: 5}, {RoleID: 1, MenuID: 4, OperationID: 6},
		{RoleID: 1, MenuID: 5, OperationID: 7}, {RoleID: 1, MenuID: 5, OperationID: 8},
		{RoleID: 1, MenuID: 5, OperationID: 9}, {RoleID: 1, MenuID: 5, OperationID: 10},
		{RoleID: 1, MenuID: 6, OperationID: 11}, {RoleID: 1, MenuID: 6, OperationID: 12},
		{RoleID: 1, MenuID: 6, OperationID: 13}, {RoleID: 1, MenuID: 6, OperationID: 14},
		{RoleID: 1, MenuID: 7, OperationID: 15}, {RoleID: 1, MenuID: 7, OperationID: 16},
		{RoleID: 1, MenuID: 7, OperationID: 17}, {RoleID: 1, MenuID: 7, OperationID: 18},
		// 测试员:权限分配(只 view + edit) + 角色管理(只 view + edit)
		{RoleID: 2, MenuID: 4, OperationID: 5}, {RoleID: 2, MenuID: 4, OperationID: 6},
		{RoleID: 2, MenuID: 7, OperationID: 15}, {RoleID: 2, MenuID: 7, OperationID: 17},
	}
	if err := db.Create(&roleOps).Error; err != nil {
		log.Fatalf("插入角色-操作失败: %v", err)
	}
	fmt.Println("✓ 角色-操作:超级管理员 14 个 / 测试员 2 个")

	fmt.Println("\n🎉 Seed 完成!")
	fmt.Println("   admin / 123456  → 超级管理员(全权限)")
	fmt.Println("   test2 / 123456  → 测试员(只有角色权限分配)")
	fmt.Println("\n下一步:重启后端 + 前端,登录 admin 验证。")
}
