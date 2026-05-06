package br

import "github.com/gin-gonic/gin"

type Br struct {
	Code    int    `json:"code"`
	Status  int    `json:"status"`
	ExtData any    `json:"extData"`
	Data    any    `json:"data"`
	Msg     string `json:"msg"`
}

var Brs = &Br{}

// 错误返回
func (b *Br) Err(c *gin.Context, msg string) {
	res := Br{Code: -1, Status: -1, Data: true, Msg: msg}
	c.JSON(200, res)
}

// 无数据成功返回
func (b *Br) Ok(c *gin.Context) {
	res := Br{Code: 1, Status: 1, Data: nil, Msg: ""}
	c.JSON(200, res)
}

// 字符串成功返回
func (b *Br) Oks(c *gin.Context, data string) {
	res := Br{Code: 1, Status: 1, Data: data, Msg: ""}
	c.JSON(200, res)
}

// 布尔成功返回
func (b *Br) Okb(c *gin.Context, data bool) {
	res := Br{Code: 1, Status: 1, Data: data, Msg: ""}
	c.JSON(200, res)
}

// 对象成功返回
func (b *Br) Oko(c *gin.Context, data any) {
	res := Br{Code: 1, Status: 1, Data: data, Msg: ""}
	c.JSON(200, res)
}

// 带扩展数据的成功返回
func (b *Br) Oke(c *gin.Context, data any, ext interface{}) {
	res := Br{Code: 1, Status: 1, ExtData: ext, Data: data, Msg: ""}
	c.JSON(200, res)
}

// 带code信息
func (b *Br) Okc(c *gin.Context, code int, msg string) {
	res := Br{Code: code, Status: code, Msg: msg}
	c.JSON(200, res)
}
