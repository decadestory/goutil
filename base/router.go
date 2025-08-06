package base

import (
	"github.com/decadestory/goutil/conf"
	"github.com/gin-gonic/gin"
)

var Router *gin.Engine

func init() {
	Router = gin.Default()

	Router.POST("/heartbeat/check", func(c *gin.Context) {
		c.JSON(200, "hello")
	})

	Router.GET("/health", func(c *gin.Context) {
		c.JSON(200, "OK")
	})

	Router.GET("/flushConfig", conf.Configs.FlushConfig)
}
