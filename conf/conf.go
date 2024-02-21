package conf

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/Unknwon/goconfig"
	"github.com/decadestory/goutil/exception"
	"github.com/spf13/viper"
)

type Config struct {
	Env string
}

var Configs = &Config{Env: ""}

func init() {

	flag.StringVar(&Configs.Env, "env", "", "-env [环境名称] 默认无")
	flag.Parse()

	confile := "conf"

	if Configs.Env != "" {
		confile += "_" + Configs.Env
	}

	viper.SetConfigType("toml")
	viper.SetConfigName(confile)
	viper.AddConfigPath(Configs.GetWorkDir() + "/conf/") // optionally look for config in the working directory
	err := viper.ReadInConfig()                          // Find and read the config file
	exception.Errors.CheckErr(err)
}

func (cfg *Config) Init() {}

func (cfg *Config) GetWorkDir() string {
	workDir := os.Getenv("DOCKER_GO_WORK_DIR")
	if workDir != "" {
		return workDir
	}

	executablePath, _ := os.Executable()
	return filepath.Dir(executablePath)
}

// 根据key获取配置文件中的值
func (cfg *Config) GetAppConf(configName string) string {
	c, err := goconfig.LoadConfigFile("conf/conf.ini")
	exception.Errors.CheckErr(err)
	res, err := c.GetValue("DEFAULT", configName)
	exception.Errors.CheckErr(err)
	return res
}

// [toml]根据key获取string
func (cfg *Config) GetString(configName string) string {
	return viper.GetString(configName)
}

// [toml]根据key获取bool
func (cfg *Config) GetBool(configName string) bool {
	return viper.GetBool(configName)
}

// [toml]根据key获取int
func (cfg *Config) GetInt(configName string) int {
	return viper.GetInt(configName)
}

// [toml]根据key获取Viper对象
func (cfg *Config) Viper() *viper.Viper {
	return viper.GetViper()
}
