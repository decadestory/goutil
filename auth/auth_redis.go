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
		AuthRds.Expire = time.Duration(effTime) * 60 * time.Minute
	}

	ignore := conf.Configs.Viper().Get("auth.ignore")
	if ignore != nil {
		AuthRds.Ignore = []string{}
		for _, each := range ignore.([]interface{}) {
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
		br.Brs.Okc(c, "认证失败，请重新登录", 403)
		c.Abort()
		return
	}

	userJson := redis.Rdb["default"].Get(c, token)
	if userJson.Err() != nil {
		br.Brs.Okc(c, "认证失败，请重新登录", 403)
		c.Abort()
		return
	}

	// TODO: 这里可以添加对token的验证逻辑，比如签名验证等
	var curUser misc.LoginResult
	json.Unmarshal([]byte(userJson.Val()), &curUser)

	//验证token，并存储在请求中
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
