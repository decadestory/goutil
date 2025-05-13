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

	Router.GET("/flushConfig", conf.Configs.FlushConfig)
}
