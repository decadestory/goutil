package redis

import (
	"strings"

	"github.com/decadestory/goutil/conf"
	"github.com/decadestory/goutil/exception"
	"github.com/redis/go-redis/v9"
)

var Rdb map[string]*redis.Client = make(map[string]*redis.Client)
var Rdbc map[string]*redis.ClusterClient = make(map[string]*redis.ClusterClient)

// StringCmd 是 go-redis StringCmd 的别名，调用方无需再直接引入 go-redis
type StringCmd = redis.StringCmd

// Cmd 是 go-redis Cmd 的别名，用于 Expire/Set 等不返回具体类型的命令
type Cmd = redis.Cmd

func init() {

	defer exception.Errors.DeferRecover()

	redisArr := conf.Configs.Viper().Get("redis")
	if redisArr != nil {
		for _, v := range redisArr.([]interface{}) {
			mv := v.(map[string]interface{})
			dbNo := 0
			usr := ""
			name := "default"

			if vdb, ok := mv["db"]; ok {
				dbNo = int(vdb.(int64))
			}

			if vdb, ok := mv["user"]; ok {
				usr = vdb.(string)
			}

			if vdb, ok := mv["name"]; ok {
				name = vdb.(string)
			}

			Rdb[name] = redis.NewClient(&redis.Options{
				Addr:     mv["host"].(string),
				Username: usr,
				Password: mv["pwd"].(string),
				DB:       dbNo,
			})
		}
	}

	redisClusterArr := conf.Configs.Viper().Get("redis-cluster")
	if redisClusterArr != nil {
		for _, v := range redisClusterArr.([]interface{}) {
			mv := v.(map[string]interface{})
			usr := ""
			name := "default"

			if vdb, ok := mv["user"]; ok {
				usr = vdb.(string)
			}

			if vdb, ok := mv["name"]; ok {
				name = vdb.(string)
			}

			Rdbc[name] = redis.NewClusterClient(&redis.ClusterOptions{
				Addrs:    strings.Split(mv["host"].(string), ","),
				Username: usr,
				Password: mv["pwd"].(string),
			})
		}
	}

}
