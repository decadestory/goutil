package auth

import (
	"encoding/json"
	"time"

	"github.com/decadestory/goutil/br"
	"github.com/decadestory/goutil/conf"
	"github.com/decadestory/goutil/misc"
	"github.com/decadestory/goutil/redis"
	"github.com/gin-gonic/gin"
)

type authRd struct {
	//该路由下不校验token
	Ignore []string
	//token有效时间
	Expire time.Duration
	//token名称
	Name string
}

var AuthRds = &authRd{
	Ignore: []string{"/health", "/ui", "/swagger", "/signIn", "/sendVaildCode", "/comm"},
	Expire: 8 * 60 * time.Minute,
	Name:   "token",
}

func init() {
	effTime := conf.Configs.GetInt("auth.expire")
	if effTime > 0 {
		AuthRds.Expire = time.Duration(effTime) * time.Minute
	}

	ignore := conf.Configs.Viper().Get("auth.ignore")
	if ignore != nil {
		AuthRds.Ignore = []string{}
		for _, each := range ignore.([]any) {
			AuthRds.Ignore = append(AuthRds.Ignore, each.(string))
		}
	}

	name := conf.Configs.Viper().Get("auth.name")
	if name != nil {
		AuthRds.Name = name.(string)
	}
}

func (a *authRd) AuthMiddleware(c *gin.Context) {

	//过滤是否验证token
	if misc.IsItemLike(a.Ignore, c.Request.RequestURI) {
		return
	}

	token := c.GetHeader(a.Name)
	if token == "" {
		br.Brs.Okc(c, 403, "认证失败，请重新登录")
		c.Abort()
		return
	}

	userJson := redis.Rdb["default"].Get(c, token)
	if userJson.Err() != nil {
		br.Brs.Okc(c, 403, "认证失败，请重新登录")
		c.Abort()
		return
	}

	// 解析token
	var curUser misc.LoginResult
	json.Unmarshal([]byte(userJson.Val()), &curUser)

	// 更新过期时间
	isRefresh := conf.Configs.GetBool("auth.refresh")
	if isRefresh {
		redis.Rdb["default"].Expire(c, token, a.Expire)
	}

	//存储在请求中
	c.Set("curUser", curUser)
}

func (a *authRd) GetCurUser(c *gin.Context) misc.LoginResult {
	claims, ok := c.Get("curUser")
	if !ok {
		uc := misc.LoginResult{}
		return uc
	}
	return claims.(misc.LoginResult)
}
