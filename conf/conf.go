package conf

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/Unknwon/goconfig"
	"github.com/decadestory/goutil/br"
	"github.com/decadestory/goutil/exception"
	"github.com/decadestory/goutil/misc"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type Config struct {
	Env string // 环境名称
	Svc string // 服务名称
	Cch string // 配置中心地址IP:端口
}

var Configs = &Config{Env: "", Svc: "", Cch: ""}

func init() {

	flag.StringVar(&Configs.Env, "env", "", "-env [环境名称] 默认无")
	flag.StringVar(&Configs.Svc, "svc", "", "-svc [服务名称] 默认无")
	flag.StringVar(&Configs.Cch, "cch", "", "-cch [配置中心IP] 默认无")
	flag.Parse()

	confile := "conf"

	if Configs.Env != "" {
		confile += "_" + Configs.Env
	}

	// 如果有配置中心则走配置中心
	if Configs.Env != "" && Configs.Svc != "" && Configs.Cch != "" {
		createConfigFile(Configs)
	}

	viper.SetConfigType("toml")
	viper.SetConfigName(confile)
	viper.AddConfigPath(Configs.GetWorkDir() + "/conf/") // optionally look for config in the working directory
	err := viper.ReadInConfig()                          // Find and read the config file
	exception.Errors.CheckErr(err)
}

func (cfg *Config) Init() {}

func createConfigFile(c *Config) {

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			debug.PrintStack()
		}
	}()

	params := map[string]string{
		"env":      c.Env,
		"app":      c.Svc,
		"clientIp": misc.GetIp(),
	}

	pstr, err := json.Marshal(params)
	exception.Errors.CheckErr(err)
	var httpClient = &http.Client{Timeout: time.Second * 3}
	resp, err := httpClient.Post(c.Cch+"/liuer/api/v1/getByEnvSvc", "application/json", bytes.NewBuffer(pstr))
	exception.Errors.CheckErr(err)
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	exception.Errors.CheckErr(err)
	fmt.Println(string(body))

	var res br.Br
	err = json.Unmarshal(body, &res)
	exception.Errors.CheckErr(err)
	if res.Status != 1 {
		exception.Errors.CheckErr(errors.New(res.Msg))
	}
	resultData := res.Data.(map[string]interface{})
	configData := resultData["config"].(string)

	// 写入配置文件
	cpath := Configs.GetWorkDir() + "/conf/conf_" + c.Env + ".toml"

	if _, err := os.Stat(cpath); err == nil {
		//删除配置文件
		os.Remove(cpath)
	}

	f, err := os.Create(cpath)
	exception.Errors.CheckErr(err)
	defer f.Close()
	_, err = f.Write([]byte(configData))
	exception.Errors.CheckErr(err)
}

func (cfg *Config) FlushConfig(c *gin.Context) {
	createConfigFile(cfg)
	br.Brs.Oks(c, "刷新配置成功")
}

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
