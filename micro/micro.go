package micro

import (
	"fmt"
	"log"

	"github.com/decadestory/goutil/conf"
	"github.com/decadestory/goutil/exception"
	"github.com/decadestory/goutil/misc"
	"github.com/hashicorp/consul/api"
)

type Micro struct {
	Client *api.Client
}

var MicroApi = Micro{}

// 注册服务
func (m *Micro) RegisterService() {
	rcUrl := conf.Configs.GetString("register.center.url")
	if rcUrl == "" {
		panic("Register center URL is not configured")
	}

	svcName := conf.Configs.GetString("service.name")
	if svcName == "" {
		panic("Service name is not configured")
	}

	svcPort := conf.Configs.GetInt("service.port")
	if svcPort == 0 {
		panic("Service port is not configured")
	}

	// 注册到 Consul
	consulConfig := api.DefaultConfig()
	consulConfig.Address = rcUrl
	err := error(nil)
	m.Client, err = api.NewClient(consulConfig)
	exception.Errors.Panic(err)

	localIp := misc.GetIp()
	reg := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s:%s:%d", svcName, localIp, svcPort),
		Name:    svcName,
		Address: localIp, // 本地 IP
		Port:    svcPort,
		Check: &api.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://%s:%d/health", localIp, svcPort),
			Interval: "10s",
			Timeout:  "1s",
		},
	}

	err = m.Client.Agent().ServiceRegister(reg)
	exception.Errors.Panic(err)

	log.Println("Service registered with Consul")
}

// 获取配置
func (m *Micro) CC(k string) *api.KV {
	kv := m.Client.KV()
	return kv
}

//调用服务
