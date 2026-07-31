package main
import (
  "fmt"
  "gorm.io/driver/mysql"
  "gorm.io/gorm"
)
func main() {
  dsn := "root:123456@tcp(192.168.186.200:3306)/go_server?charset=utf8mb4&parseTime=True&loc=Local"
  db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
  type RM struct { RoleID uint; MenuID uint }
  var rms []RM
  db.Raw("SELECT role_id, menu_id FROM go_admin_role_menus WHERE role_id = 1 ORDER BY menu_id").Scan(&rms)
  for _, r := range rms { fmt.Printf("  role=%d menu=%d\n", r.RoleID, r.MenuID) }
}
