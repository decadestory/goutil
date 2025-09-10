package auth

import (
	"time"

	"github.com/decadestory/goutil/br"
	"github.com/decadestory/goutil/conf"
	"github.com/decadestory/goutil/exception"
	"github.com/decadestory/goutil/misc"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

type auth struct {
	//自定义的token秘钥
	Secret []byte
	//该路由下不校验token
	Ignore []string
	//token有效时间
	Expire time.Duration
	//token名称
	Name string
}

var Auths = &auth{
	Secret: []byte("26849841325189456f487"),
	Ignore: []string{"/ui", "/res", "/export", "/favicon.ico", "/flushConfig"},
	Expire: 8 * 60 * time.Minute,
	Name:   "token",
}

// 用户信息类，作为生成token的参数
type UserClaims struct {
	UserId    int      `json:"userId"`
	UserName  string   `json:"userName"`
	RoleCodes []string `json:"roleCodes"`
	//jwt-go提供的标准claim
	jwt.StandardClaims
}

func init() {
	effTime := conf.Configs.GetInt("auth.expire")
	if effTime > 0 {
		Auths.Expire = time.Duration(effTime) * 60 * time.Minute
	}

	ignore := conf.Configs.Viper().Get("auth.ignore")
	if ignore != nil {
		Auths.Ignore = []string{}
		for _, each := range ignore.([]interface{}) {
			Auths.Ignore = append(Auths.Ignore, each.(string))
		}
	}

	name := conf.Configs.Viper().Get("auth.name")
	if name != nil {
		Auths.Name = name.(string)
	}
}

// 生成token
func (a *auth) GenerateToken(claims *UserClaims) string {
	claims.ExpiresAt = time.Now().Add(a.Expire).Unix()
	sign, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.Secret)
	exception.Errors.Panic(err)
	return sign
}

// 验证token
func (a *auth) AuthMiddleware(c *gin.Context) {

	//过滤是否验证token
	if misc.IsItemLike(a.Ignore, c.Request.RequestURI) {
		return
	}

	token := c.GetHeader(a.Name)
	if token == "" {
		br.Brs.Okc(c, "登录过期，请重新登录", 403)
		c.Abort()
		return
	}

	//解析token
	jwt, err := jwt.ParseWithClaims(token, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return a.Secret, nil
	})

	if err != nil {
		br.Brs.Okc(c, "登录过期，请重新登录", 403)
		c.Abort()
		return
	}

	claims, ok := jwt.Claims.(*UserClaims)
	if !ok {
		br.Brs.Okc(c, "登录过期，请重新登录", 403)
		c.Abort()
		return
	}

	//验证token，并存储在请求中
	c.Set("curUser", claims)
}

func (a *auth) GetCurUser(c *gin.Context) *UserClaims {
	claims, ok := c.Get("curUser")
	if !ok {
		uc := UserClaims{}
		return &uc
	}
	return claims.(*UserClaims)
}
