package service

import (
	"testing"

	"go_server/internal/model"
)

// TestBuildMenuTree_Empty 空输入返回 nil
func TestBuildMenuTree_Empty(t *testing.T) {
	got := BuildMenuTree(nil)
	if got != nil {
		// 允许是空切片,只要不是非空就行
		if len(got) != 0 {
			t.Fatalf("expected empty, got %d roots", len(got))
		}
	}
}

// TestBuildMenuTree_SingleRoot 只有一个根节点
func TestBuildMenuTree_SingleRoot(t *testing.T) {
	menus := []model.AdminMenus{
		{BaseModel: model.BaseModel{ID: 1}, Name: "根菜单", ParentID: 0},
	}
	roots := BuildMenuTree(menus)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if roots[0].Name != "根菜单" {
		t.Fatalf("expected name=根菜单, got %s", roots[0].Name)
	}
	if len(roots[0].Children) != 0 {
		t.Fatalf("expected 0 children, got %d", len(roots[0].Children))
	}
}

// TestBuildMenuTree_ParentChild 父子菜单挂载关系正确
func TestBuildMenuTree_ParentChild(t *testing.T) {
	menus := []model.AdminMenus{
		{BaseModel: model.BaseModel{ID: 1}, Name: "系统管理", Path: "/system", ParentID: 0},
		{BaseModel: model.BaseModel{ID: 2}, Name: "用户管理", Path: "/user", ParentID: 1},
		{BaseModel: model.BaseModel{ID: 3}, Name: "角色管理", Path: "/role", ParentID: 1},
		{BaseModel: model.BaseModel{ID: 4}, Name: "菜单管理", Path: "/menu", ParentID: 1},
	}
	roots := BuildMenuTree(menus)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}
	if roots[0].ID != 1 {
		t.Fatalf("expected root ID=1, got %d", roots[0].ID)
	}
	if len(roots[0].Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(roots[0].Children))
	}
	childNames := map[string]bool{}
	for _, c := range roots[0].Children {
		childNames[c.Name] = true
	}
	for _, want := range []string{"用户管理", "角色管理", "菜单管理"} {
		if !childNames[want] {
			t.Errorf("missing child %s", want)
		}
	}
}

// TestBuildMenuTree_OrphanParent 找不到 parent 时挂在根上(与原 handler 行为一致)
func TestBuildMenuTree_OrphanParent(t *testing.T) {
	menus := []model.AdminMenus{
		{BaseModel: model.BaseModel{ID: 5}, Name: "孤儿", Path: "/orphan", ParentID: 99},
	}
	roots := BuildMenuTree(menus)
	if len(roots) != 1 {
		t.Fatalf("expected 1 root (orphan treated as root), got %d", len(roots))
	}
	if roots[0].ID != 5 {
		t.Fatalf("expected root ID=5, got %d", roots[0].ID)
	}
}
