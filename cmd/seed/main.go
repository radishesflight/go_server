// Package main seed 工具 - 初始化基础数据
//
// 跑法:go run cmd/seed/main.go
// 会:
//  1. 清空所有业务表
//  2. 插入种子数据:部门 / 角色 / 菜单 / routes(API 权限) / 用户 / 角色-菜单 / 角色-路由
//
// 种子账号:
//
//	admin / 123456(超级管理员,所有权限)
//	test2 / 123456(测试员,只有"权限分配"+"角色管理"菜单的查看/编辑权限)
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

	// 2. Drop & Recreate 所有业务表(按外键依赖倒序)
	//    schema 可能跟新 model 不一致(比如旧表的 code 列还在),硬删表才能保证新 schema
	//    先 drop 全部,再用 model 重新 create
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
		if err := db.Migrator().DropTable(t); err != nil {
			log.Printf("drop 表 %T 失败(可能表不存在,跳过): %v", t, err)
		}
	}
	for _, t := range tables {
		if err := db.AutoMigrate(t); err != nil {
			log.Fatalf("create 表 %T 失败: %v", t, err)
		}
	}
	fmt.Println("✓ drop + recreate 全部业务表(用最新 schema)")

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
	fmt.Println("✓ 菜单:7 个(首页 / 系统管理 / 5 个子菜单)")

	// 6. routes(API 权限)
	//    每行 = 一个具体接口,中间件直接用 (method, path) 匹配
	routes := []model.AdminMenuOperations{
		// 菜单 menu_id=3 (管理员管理 adminUsers)
		{BaseModel: model.BaseModel{ID: 1, CreatedAt: now, UpdatedAt: now}, MenuID: 3, Method: "GET", Path: "/api/system/adminUsers/list", Name: "管理员列表", Sort: 1},
		{BaseModel: model.BaseModel{ID: 2, CreatedAt: now, UpdatedAt: now}, MenuID: 3, Method: "GET", Path: "/api/system/adminUsers/:id", Name: "管理员详情", Sort: 2},
		{BaseModel: model.BaseModel{ID: 3, CreatedAt: now, UpdatedAt: now}, MenuID: 3, Method: "POST", Path: "/api/system/adminUsers", Name: "新增管理员", Sort: 3},
		{BaseModel: model.BaseModel{ID: 4, CreatedAt: now, UpdatedAt: now}, MenuID: 3, Method: "PUT", Path: "/api/system/adminUsers/:id", Name: "编辑管理员", Sort: 4},
		{BaseModel: model.BaseModel{ID: 5, CreatedAt: now, UpdatedAt: now}, MenuID: 3, Method: "DELETE", Path: "/api/system/adminUsers/:id", Name: "删除管理员", Sort: 5},

		// 菜单 menu_id=4 (权限分配 roleMenu)
		{BaseModel: model.BaseModel{ID: 6, CreatedAt: now, UpdatedAt: now}, MenuID: 4, Method: "GET", Path: "/api/system/roleMenu/allMenus", Name: "所有菜单", Sort: 1},
		{BaseModel: model.BaseModel{ID: 7, CreatedAt: now, UpdatedAt: now}, MenuID: 4, Method: "GET", Path: "/api/system/roleMenu/roleMenus", Name: "角色已分配菜单", Sort: 2},
		{BaseModel: model.BaseModel{ID: 8, CreatedAt: now, UpdatedAt: now}, MenuID: 4, Method: "GET", Path: "/api/system/roleMenu/roleRoutes", Name: "角色已分配路由", Sort: 3},
		{BaseModel: model.BaseModel{ID: 9, CreatedAt: now, UpdatedAt: now}, MenuID: 4, Method: "PUT", Path: "/api/system/roleMenu/assign", Name: "保存权限分配", Sort: 4},

		// 菜单 menu_id=5 (菜单管理 adminMenus)
		{BaseModel: model.BaseModel{ID: 10, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Method: "GET", Path: "/api/system/adminMenus/list", Name: "菜单列表", Sort: 1},
		{BaseModel: model.BaseModel{ID: 11, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Method: "GET", Path: "/api/system/adminMenus/all", Name: "所有菜单(带路由)", Sort: 2},
		{BaseModel: model.BaseModel{ID: 12, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Method: "GET", Path: "/api/system/adminMenus/options", Name: "父级菜单下拉", Sort: 3},
		{BaseModel: model.BaseModel{ID: 13, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Method: "GET", Path: "/api/system/adminMenus/operations/:menu_id", Name: "某菜单的路由", Sort: 4},
		{BaseModel: model.BaseModel{ID: 14, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Method: "GET", Path: "/api/system/adminMenus/:id", Name: "菜单详情", Sort: 5},
		{BaseModel: model.BaseModel{ID: 15, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Method: "POST", Path: "/api/system/adminMenus", Name: "新增菜单", Sort: 6},
		{BaseModel: model.BaseModel{ID: 16, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Method: "PUT", Path: "/api/system/adminMenus/:id", Name: "编辑菜单", Sort: 7},
		{BaseModel: model.BaseModel{ID: 17, CreatedAt: now, UpdatedAt: now}, MenuID: 5, Method: "DELETE", Path: "/api/system/adminMenus/:id", Name: "删除菜单", Sort: 8},

		// 菜单 menu_id=6 (部门管理 departments)
		{BaseModel: model.BaseModel{ID: 18, CreatedAt: now, UpdatedAt: now}, MenuID: 6, Method: "GET", Path: "/api/system/departments", Name: "部门列表", Sort: 1},

		// 菜单 menu_id=7 (角色管理 adminRoles)
		{BaseModel: model.BaseModel{ID: 19, CreatedAt: now, UpdatedAt: now}, MenuID: 7, Method: "GET", Path: "/api/system/adminRoles/list", Name: "角色列表", Sort: 1},
		{BaseModel: model.BaseModel{ID: 20, CreatedAt: now, UpdatedAt: now}, MenuID: 7, Method: "GET", Path: "/api/system/adminRoles/:id", Name: "角色详情", Sort: 2},
		{BaseModel: model.BaseModel{ID: 21, CreatedAt: now, UpdatedAt: now}, MenuID: 7, Method: "POST", Path: "/api/system/adminRoles", Name: "新增角色", Sort: 3},
		{BaseModel: model.BaseModel{ID: 22, CreatedAt: now, UpdatedAt: now}, MenuID: 7, Method: "PUT", Path: "/api/system/adminRoles/:id", Name: "编辑角色", Sort: 4},
		{BaseModel: model.BaseModel{ID: 23, CreatedAt: now, UpdatedAt: now}, MenuID: 7, Method: "DELETE", Path: "/api/system/adminRoles/:id", Name: "删除角色", Sort: 5},
	}
	if err := db.Create(&routes).Error; err != nil {
		log.Fatalf("插入路由失败: %v", err)
	}
	fmt.Printf("✓ 路由:23 条(每个 API 接口一条)\n")

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

	// 8. 角色-菜单(超级管理员:全部,测试员:首页+系统管理+角色+权限分配)
	roleMenus := []model.AdminRoleMenus{
		// 超级管理员
		{RoleID: 1, MenuID: 1}, {RoleID: 1, MenuID: 2}, {RoleID: 1, MenuID: 3},
		{RoleID: 1, MenuID: 4}, {RoleID: 1, MenuID: 5}, {RoleID: 1, MenuID: 6},
		{RoleID: 1, MenuID: 7},
		// 测试员:首页 + 系统管理 + 角色管理 + 权限分配
		{RoleID: 2, MenuID: 1}, {RoleID: 2, MenuID: 2}, {RoleID: 2, MenuID: 4},
		{RoleID: 2, MenuID: 7},
	}
	if err := db.Create(&roleMenus).Error; err != nil {
		log.Fatalf("插入角色-菜单失败: %v", err)
	}
	fmt.Println("✓ 角色-菜单:超级管理员 7 个 / 测试员 4 个")

	// 9. 角色-路由(基于真实 API 列表)
	//    超级管理员:全部 23 条
	//    测试员:角色管理 view+edit + 权限分配 view+edit
	//      roleMenus: GET 19, GET 20(view only), GET 6, 7, 8
	//      roleAssign: PUT 9(edit)
	//      roles  edit: PUT 22
	roleOps := []model.AdminRoleOperations{
		// 超级管理员:全部
		{RoleID: 1, RouteID: 1}, {RoleID: 1, RouteID: 2}, {RoleID: 1, RouteID: 3},
		{RoleID: 1, RouteID: 4}, {RoleID: 1, RouteID: 5},
		{RoleID: 1, RouteID: 6}, {RoleID: 1, RouteID: 7}, {RoleID: 1, RouteID: 8},
		{RoleID: 1, RouteID: 9},
		{RoleID: 1, RouteID: 10}, {RoleID: 1, RouteID: 11}, {RoleID: 1, RouteID: 12},
		{RoleID: 1, RouteID: 13}, {RoleID: 1, RouteID: 14}, {RoleID: 1, RouteID: 15},
		{RoleID: 1, RouteID: 16}, {RoleID: 1, RouteID: 17},
		{RoleID: 1, RouteID: 18},
		{RoleID: 1, RouteID: 19}, {RoleID: 1, RouteID: 20}, {RoleID: 1, RouteID: 21},
		{RoleID: 1, RouteID: 22}, {RoleID: 1, RouteID: 23},

		// 测试员:权限分配(查看 + 保存)+ 角色管理(查看 + 编辑)
		{RoleID: 2, RouteID: 6},  // GET /api/system/roleMenu/allMenus
		{RoleID: 2, RouteID: 7},  // GET /api/system/roleMenu/roleMenus
		{RoleID: 2, RouteID: 8},  // GET /api/system/roleMenu/roleRoutes
		{RoleID: 2, RouteID: 9},  // PUT /api/system/roleMenu/assign
		{RoleID: 2, RouteID: 19}, // GET /api/system/adminRoles/list
		{RoleID: 2, RouteID: 20}, // GET /api/system/adminRoles/:id
		{RoleID: 2, RouteID: 22}, // PUT /api/system/adminRoles/:id
	}
	if err := db.Create(&roleOps).Error; err != nil {
		log.Fatalf("插入角色-路由失败: %v", err)
	}
	fmt.Println("✓ 角色-路由:超级管理员 23 个 / 测试员 7 个")

	fmt.Println("\n🎉 Seed 完成!")
	fmt.Println("   admin / 123456  → 超级管理员(全权限)")
	fmt.Println("   test2 / 123456  → 测试员(角色管理 + 权限分配)")
	fmt.Println("\n下一步:重启后端 + 前端,登录 admin 验证。")
}
