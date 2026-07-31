package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
// 业务码模式:HTTP 状态码恒为 200,真实状态在 body.code 里
//   code 字段语义:
//     0    成功
//     非 0  业务错误(见 bizcode.go)
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  "success",
		Data: data,
	})
}

// Error 错误响应
// code 必须是 handler.CodeXxx 业务码
// msg  给用户看的错误文案
func Error(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  msg,
	})
}
