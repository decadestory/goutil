package micro

import (
	"encoding/json"
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

// 获取配置
func (m *Micro) CC() *api.KV {
	kv := m.Client.KV()
	return kv
}

func (m *Micro) CStr(key string) string {
	kv := m.Client.KV()
	pair, _, _ := kv.Get(key, nil)
	return string(pair.Value)
}

// ["192.168.1.10", "192.168.1.11", "192.168.1.12"]
func (m *Micro) CStrArr(key string) []string {
	kv := m.Client.KV()
	pair, _, _ := kv.Get(key, nil)
	var arr []string
	json.Unmarshal(pair.Value, &arr)
	return arr
}

func (m *Micro) CSvcCache(key_prefix string) {
	kv := m.Client.KV()
	pairs, _, _ := kv.List(key_prefix, nil)
	m.Config = &pairs
}

//调用服务
