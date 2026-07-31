// Package service menu_tree.go - 菜单树构建工具
//
// 纯函数工具,不依赖 DB,放在 service 包方便 service.AuthService 复用
// 也有完整的单元测试(internal/service/menu_tree_test.go)
package service

import (
	"strings"

	"go_server/internal/model"
)

// MenuTreeNode 菜单树节点
// 字段名 / JSON tag 与前端期望一致
type MenuTreeNode struct {
	ID       uint            `json:"id"`
	Name     string          `json:"name"`
	Path     string          `json:"path"`
	Icon     string          `json:"icon"`
	ParentID uint            `json:"parent_id"`
	Children []*MenuTreeNode `json:"children"`
}

// BuildMenuTree 把扁平菜单列表构造成树
// 行为细节:
//  1. 第一遍遍历建 map(id → node)
//  2. 第二遍遍历建父子关系
//  3. parent_id == 0 是根
//  4. parent_id 找不到对应节点(孤儿) → 当作根
//  5. 父节点 path 拼到子节点 path 前面(继承路由前缀)
//
// 注意:此函数会**修改**输入的 path 字段(给子节点拼父节点路径)
func BuildMenuTree(menus []model.AdminMenus) []*MenuTreeNode {
	nodeMap := make(map[uint]*MenuTreeNode)
	var roots []*MenuTreeNode

	for _, m := range menus {
		nodeMap[m.ID] = &MenuTreeNode{
			ID:       m.ID,
			Name:     m.Name,
			Path:     m.Path,
			Icon:     m.Icon,
			ParentID: m.ParentID,
			Children: make([]*MenuTreeNode, 0),
		}
	}

	for _, m := range menus {
		node := nodeMap[m.ID]
		if m.ParentID == 0 {
			roots = append(roots, node)
		} else {
			if parent, ok := nodeMap[m.ParentID]; ok {
				// 子 path 处理:
				//   - 如果是绝对路径(以 / 开头)→ 保持原样(已经含完整路由,如 "/system/adminUsers")
				//   - 如果是相对路径 → 拼上父 path(如 "users/list" → "/system/users/list")
				if strings.HasPrefix(node.Path, "/") {
					// 绝对路径,直接用,不要再拼父 path
				} else {
					node.Path = strings.TrimRight(parent.Path, "/") + "/" + strings.TrimLeft(node.Path, "/")
				}
				parent.Children = append(parent.Children, node)
			} else {
				// 孤儿节点(找不到 parent)挂根上
				roots = append(roots, node)
			}
		}
	}

	return roots
}
