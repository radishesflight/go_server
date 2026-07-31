package config

import "testing"

func TestCORSConfig_SplitOrigins_Empty(t *testing.T) {
	c := CORSConfig{AllowedOrigins: ""}
	got := c.SplitOrigins()
	if len(got) != 1 || got[0] != "*" {
		t.Fatalf("expected [*], got %v", got)
	}
}

func TestCORSConfig_SplitOrigins_Wildcard(t *testing.T) {
	c := CORSConfig{AllowedOrigins: "*"}
	got := c.SplitOrigins()
	if len(got) != 1 || got[0] != "*" {
		t.Fatalf("expected [*], got %v", got)
	}
}

func TestCORSConfig_SplitOrigins_List(t *testing.T) {
	c := CORSConfig{AllowedOrigins: "http://localhost:5173, https://admin.example.com ,, http://b.com"}
	got := c.SplitOrigins()
	want := []string{"http://localhost:5173", "https://admin.example.com", "http://b.com"}
	if len(got) != len(want) {
		t.Fatalf("expected %d origins, got %d (%v)", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("origin[%d]: want %s, got %s", i, w, got[i])
		}
	}
}
