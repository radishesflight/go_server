package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Success(c, gin.H{"foo": "bar"})

	if w.Code != 200 {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}
	var got Response
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Code != CodeSuccess {
		t.Errorf("expected code=%d, got %d", CodeSuccess, got.Code)
	}
	if got.Msg != "success" {
		t.Errorf("expected msg=success, got %s", got.Msg)
	}
	if got.Data == nil {
		t.Errorf("expected data non-nil")
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Error(c, CodeUserNotFound, "用户不存在")

	if w.Code != 200 {
		t.Fatalf("expected HTTP 200, got %d", w.Code)
	}
	var got Response
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Code != CodeUserNotFound {
		t.Errorf("expected code=%d, got %d", CodeUserNotFound, got.Code)
	}
	if got.Msg != "用户不存在" {
		t.Errorf("expected msg=用户不存在, got %s", got.Msg)
	}
	if got.Data != nil {
		t.Errorf("expected data nil, got %v", got.Data)
	}
}

func TestError_DataOmitted(t *testing.T) {
	// Error 时 Data 字段应该被 omitempty 隐藏
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Error(c, CodeUnknown, "x")

	body := w.Body.String()
	if gotSubstr(t, body, `"data"`) {
		t.Errorf("expected data field omitted, got body: %s", body)
	}
}

func gotSubstr(t *testing.T, s, sub string) bool {
	t.Helper()
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
