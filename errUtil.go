package goutil

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Error struct{}

var Errors = &Error{}

// 检查错误，打印日志
func (e *Error) CheckErr(err error) {
	if err != nil {
		log.Println(err)
	}
}

// 检查错误，打印日志，抛出异常
func (e *Error) Panic(err error) {
	if err != nil {
		panic(err)
	}
}

// Gin中间件，全局捕获异常
func (e *Error) Recover(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic: %v\n", r)
			debug.PrintStack()
			//封装通用json返回
			res := Br{Status: -2, ExtData: 0, Data: nil, Msg: Errors.ErrorToString(r)}
			c.JSON(http.StatusOK, res)
			//终止后续接口调用，不加的话recover到异常后，还会继续执行接口里后续代码
			c.Abort()
		}
	}()
	//加载完 defer recover，继续后续接口调用
	c.Next()
}

// recover错误，转string
func (e *Error) ErrorToString(r interface{}) string {
	switch v := r.(type) {
	case error:
		return v.Error()
	default:
		return r.(string)
	}
}

// 事务中recover错误，抛出异常
func (e *Error) TranRecover(tx *gorm.DB) {
	if r := recover(); r != nil {
		tx.Rollback()
		panic(r)
	}
}
