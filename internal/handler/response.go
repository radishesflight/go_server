// Package handler 提供 HTTP 业务响应工具
//
// 所有业务接口的响应格式:
//  {
//    "code": 0,                  // 业务码(0=成功,非0=具体错误,见 bizcode.go)
//    "msg": "success",            // 给用户看的错误文案
//    "data": { ... }              // 成功时的业务数据
//  }
//
// HTTP 状态码**恒为 200**,真实状态在 body.code 里。
// 这是国内后台项目的标准做法(避免浏览器拦截 4xx、网关改写状态码)。
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// Success 成功响应
// HTTP 200 + {code: 0, msg: "success", data: ...}
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code: CodeSuccess,
		Msg:  "success",
		Data: data,
	})
}

// Error 错误响应
// HTTP 200 + {code: <业务码>, msg: <错误文案>}
// code 必须是 handler.CodeXxx 业务码(见 bizcode.go)
// msg 是给用户看的提示文案
func Error(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Response{
		Code: code,
		Msg:  msg,
	})
}
