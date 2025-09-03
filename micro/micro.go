package micro

import (
	"fmt"

	"github.com/decadestory/goutil/conf"
	"github.com/decadestory/goutil/exception"
	"github.com/decadestory/goutil/misc"
	"github.com/hashicorp/consul/api"
)

type Micro struct {
	Client *api.Client
	Config *api.KVPairs
}

var Micros = Micro{}

func init() {
	Micros.RegisterService()
}

// 注册服务
func (m *Micro) RegisterService() {
	rcUrl := conf.Configs.GetString("register.center.url")
	svcName := conf.Configs.GetString("service.name")
	svcPort := conf.Configs.GetInt("service.port")

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

	fmt.Println("Service registered successfully:", reg.ID)
}

//调用服务
