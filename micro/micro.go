package micro

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/decadestory/goutil/br"
	"github.com/decadestory/goutil/conf"
	"github.com/decadestory/goutil/exception"
	"github.com/decadestory/goutil/misc"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

type Micro struct {
	client   *api.Client
	services map[string]cacheService
	ttl      time.Duration
	mu       sync.RWMutex
	counters map[string]*uint64 // 轮询计数器
}

type cacheService struct {
	lastUpdate time.Time
	svcs       []*api.ServiceEntry
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
	m.client, err = api.NewClient(consulConfig)
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

	err = m.client.Agent().ServiceRegister(reg)
	exception.Errors.Panic(err)

	m.ttl = time.Second * 10
	m.services = make(map[string]cacheService)
	m.counters = make(map[string]*uint64)
	fmt.Println("Service registered successfully:", reg.ID)
}

// 调用服务
func (m *Micro) Invoke(c *gin.Context, serviceName, api string, param any, result *br.Br) error {
	// 获取服务实例
	service, err := m.getService(serviceName)
	if err != nil {
		return err
	}

	// 准备请求体
	var requestBody []byte
	if param != nil {
		requestBody, err = json.Marshal(param)
		if err != nil {
			return err
		}
	}

	// 创建HTTP请求
	serviceURL := fmt.Sprintf("http://%s:%d%s", service.Address, service.Port, api)
	req, err := http.NewRequest("POST", serviceURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 添加token头（如果存在）
	if token := c.GetHeader("token"); token != "" {
		req.Header.Set("token", token)
	}

	// 添加requestId头（如果存在）
	if requestId := c.GetHeader("requestId"); requestId != "" {
		req.Header.Set("requestId", requestId)
	}

	// 发送请求
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 返回响应给客户端
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)

	err = json.Unmarshal(body, result)
	if err != nil {
		return err
	}

	return nil
}

func (m *Micro) getService(serviceName string) (*api.AgentService, error) {
	m.mu.RLock()
	entries, ok := m.services[serviceName]
	expired := !ok || time.Since(entries.lastUpdate) > m.ttl
	m.mu.RUnlock()

	if expired {
		// 缓存过期，重新拉取
		svcs, _, err := m.client.Health().Service(serviceName, "", true, nil)
		if err != nil {
			return nil, err
		}

		if len(svcs) == 0 {
			return nil, fmt.Errorf("no healthy instance found for %s", serviceName)
		}
		m.mu.Lock()
		m.services[serviceName] = cacheService{svcs: svcs, lastUpdate: time.Now()}
		entries = m.services[serviceName]
		m.mu.Unlock()
	}

	if len(entries.svcs) == 0 {
		return nil, fmt.Errorf("no service instance available for %s", serviceName)
	}

	// 轮询分配，返回节点
	m.mu.Lock()
	defer m.mu.Unlock()

	// 初始化计数器（如果不存在）
	if _, exists := m.counters[serviceName]; !exists {
		var counter uint64 = 0
		m.counters[serviceName] = &counter
	}

	// 使用原子操作获取并递增计数器
	counter := m.counters[serviceName]
	index := atomic.AddUint64(counter, 1) % uint64(len(entries.svcs))
	selectedService := entries.svcs[index]

	return selectedService.Service, nil
}

// Convert 把 any 转换成指定泛型类型 T
func Convert[T any](v any) (T, error) {
	var result T

	// 先把 v 转成 JSON
	data, err := json.Marshal(v)
	if err != nil {
		return result, err
	}

	// 再反序列化到目标类型
	err = json.Unmarshal(data, &result)
	return result, err
}
