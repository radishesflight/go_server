package service

import (
	"strings"

	"go_server/internal/model"
)

// MenuTreeNode 菜单树节点
// 字段名 / JSON tag 与原 handler.MenuTreeNode 完全一致,前端无感
type MenuTreeNode struct {
	ID       uint            `json:"id"`
	Name     string          `json:"name"`
	Path     string          `json:"path"`
	Icon     string          `json:"icon"`
	ParentID uint            `json:"parent_id"`
	Children []*MenuTreeNode `json:"children"`
}

// BuildMenuTree 把扁平菜单列表构造成树
// 行为与原 handler.buildMenuTree 一致
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
				node.Path = parent.Path + strings.TrimPrefix(node.Path, "")
				parent.Children = append(parent.Children, node)
			} else {
				roots = append(roots, node)
			}
		}
	}

	return roots
}
