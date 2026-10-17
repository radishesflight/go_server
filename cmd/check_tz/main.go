package main

import (
	"fmt"
	"time"
)

func main() {
	utc, _ := time.LoadLocation("UTC")
	fmt.Printf("TZ env=%q location name=%q\n", time.Local.String(), utc.String())

	cases := []struct {
		layout string
		s      string
	}{
		// 标准 RFC3339
		{time.RFC3339, "2026-08-12T08:52Z"},
		{time.RFC3339, "2026-08-12T08:52:31Z"},
		{time.RFC3339, "2026-08-12T08:52+00:00"},
		// 手写 layout
		{"2006-01-02T15:04:05Z07:00", "2026-08-12T08:52Z"},
		{"2006-01-02T15:04:05-0700", "2026-08-12T08:52+0000"},
		{"2006-01-02T15:04:05", "2026-08-12T08:52"},
		// ParseInLocation
	}
	for _, c := range cases {
		if t, err := time.Parse(c.layout, c.s); err == nil {
			fmt.Printf("Parse OK   layout=%-30s s=%-25s -> %s\n", c.layout, c.s, t.Format(time.RFC3339))
		} else {
			fmt.Printf("Parse FAIL layout=%-30s s=%-25s -> %v\n", c.layout, c.s, err)
		}
		if t, err := time.ParseInLocation(c.layout, c.s, utc); err == nil {
			fmt.Printf("PIL   OK   layout=%-30s s=%-25s -> %s\n", c.layout, c.s, t.Format(time.RFC3339))
		} else {
			fmt.Printf("PIL   FAIL layout=%-30s s=%-25s -> %v\n", c.layout, c.s, err)
		}
	}
}
