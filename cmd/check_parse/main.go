// 临时诊断文件,验证 time.Parse 在当前 Go 环境是否能 parse "2026-08-12T08:52Z"
// 删除:cd go_server && rm -rf cmd/check_parse
package main

import (
	"fmt"
	"time"
)

func main() {
	s := "2026-08-12T08:52Z"
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	} {
		if t, err := time.Parse(layout, s); err != nil {
			fmt.Printf("[FAIL] layout=%q err=%v\n", layout, err)
		} else {
			fmt.Printf("[OK]   layout=%q -> %v\n", layout, t)
		}
	}
}
