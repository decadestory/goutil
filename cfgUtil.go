package goutil

import (
	"github.com/Unknwon/goconfig"
)

type Config struct{}

var Configs = &Config{}

// 根据key获取配置文件中的值
func (cfg *Config) GetAppConf(configName string) string {
	c, err := goconfig.LoadConfigFile("conf/conf.ini")
	Errors.CheckErr(err)
	res, _ := c.GetValue("DEFAULT", configName)
	return res
}
