package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/decadestory/goutil/br"
	"github.com/decadestory/goutil/conf"
	"github.com/decadestory/goutil/misc"
	"github.com/decadestory/goutil/redis"
	"github.com/gin-gonic/gin"
)

// 第三方应用密钥来源。业务方接 DB/Redis 时替换 AuthApis.Apps 即可，
// 未设置时走配置文件的 auth.api.app 列表。
type ApiAppLoader func(c *gin.Context, appKey string) (misc.AppInfo, bool)

type apiAuth struct {
	// 该路由下不校验签名
	Ignore []string
	// 请求头名称
	KeyName       string
	TimestampName string
	NonceName     string
	SignName      string
	// 时间戳有效窗口，超出即认为请求过期
	Expire time.Duration
	// 参与签名的请求体上限，超过则拒绝
	MaxBody int64
	// 是否校验 nonce 防重放
	NonceCheck bool
	// Redis 不可用时是否放行。true 表示拿到 nonce 就放行（签名依然校验），
	// false 表示直接拒绝，避免 Redis 故障期间被重放
	NonceFailOpen bool
	// 认证失败的返回业务码
	FailCode int
	// 密钥来源，为空则读配置
	AppFunc ApiAppLoader
}

var AuthApis = &apiAuth{
	Ignore:        []string{"/health", "/ui", "/swagger"},
	KeyName:       "X-Api-Key",
	TimestampName: "X-Timestamp",
	NonceName:     "X-Nonce",
	SignName:      "X-Sign",
	Expire:        5 * time.Minute,
	MaxBody:       4 << 20,
	NonceCheck:    true,
	FailCode:      401,
}

func init() {
	expire := conf.Configs.GetInt("auth.api.expire")
	if expire > 0 {
		AuthApis.Expire = time.Duration(expire) * time.Minute
	}

	maxBody := conf.Configs.GetInt("auth.api.maxbody")
	if maxBody > 0 {
		AuthApis.MaxBody = int64(maxBody)
	}

	failCode := conf.Configs.GetInt("auth.api.failcode")
	if failCode > 0 {
		AuthApis.FailCode = failCode
	}

	if name := conf.Configs.GetString("auth.api.keyname"); name != "" {
		AuthApis.KeyName = name
	}
	if name := conf.Configs.GetString("auth.api.timestampname"); name != "" {
		AuthApis.TimestampName = name
	}
	if name := conf.Configs.GetString("auth.api.noncename"); name != "" {
		AuthApis.NonceName = name
	}
	if name := conf.Configs.GetString("auth.api.signname"); name != "" {
		AuthApis.SignName = name
	}

	// 注意判空，否则配置项缺失时会把默认开启的 nonce 校验覆盖成关闭
	if v := conf.Configs.Viper().Get("auth.api.noncecheck"); v != nil {
		AuthApis.NonceCheck = conf.Configs.GetBool("auth.api.noncecheck")
	}
	if v := conf.Configs.Viper().Get("auth.api.noncefailopen"); v != nil {
		AuthApis.NonceFailOpen = conf.Configs.GetBool("auth.api.noncefailopen")
	}

	ignore := conf.Configs.Viper().Get("auth.api.ignore")
	if ignore != nil {
		AuthApis.Ignore = []string{}
		for _, each := range ignore.([]any) {
			AuthApis.Ignore = append(AuthApis.Ignore, each.(string))
		}
	}
}

// 第三方调用认证中间件。
//
// 请求头：
//
//	X-Api-Key   第三方标识
//	X-Timestamp 请求时间戳，秒或毫秒均可
//	X-Nonce     随机串，窗口内不可重复
//	X-Sign      hex(HMAC-SHA256(AppSecret, 规范化串))
//
// 规范化串共 6 行，以 \n 连接：
//
//	METHOD\nPATH\nSORTED_QUERY\nSHA256_HEX(BODY)\nTIMESTAMP\nNONCE
//
// 各语言签名实现见 auth/api_auth.md
func (a *apiAuth) AuthMiddleware(c *gin.Context) {

	//过滤是否验证签名
	if misc.IsItemLike(a.Ignore, c.Request.RequestURI) {
		return
	}

	appKey := c.GetHeader(a.KeyName)
	timestamp := c.GetHeader(a.TimestampName)
	nonce := c.GetHeader(a.NonceName)
	sign := c.GetHeader(a.SignName)

	if appKey == "" || timestamp == "" || nonce == "" || sign == "" {
		a.fail(c, "缺少认证参数")
		return
	}

	//校验时间戳窗口，防过期请求重放
	ts, ok := parseTimestamp(timestamp)
	if !ok {
		a.fail(c, "时间戳格式错误")
		return
	}
	if diff := time.Since(ts); diff > a.Expire || diff < -a.Expire {
		a.fail(c, "请求已过期")
		return
	}

	//查询第三方密钥
	app, ok := a.AppFunc(c, appKey)
	if !ok || !app.Valid {
		a.fail(c, "无效的 ApiKey")
		return
	}

	//读取请求体用于验签，读完后必须还原，后续业务还要用
	body, err := a.readBody(c)
	if err != nil {
		a.fail(c, "请求体读取失败")
		return
	}
	if int64(len(body)) > a.MaxBody {
		a.fail(c, "请求体过大")
		return
	}

	//验签，用 hmac.Equal 做等长常量时间比较，避免时序攻击
	expect := Sign(app.Secret, c.Request.Method, c.Request.URL.EscapedPath(),
		c.Request.URL.Query(), body, timestamp, nonce)
	if !hmac.Equal([]byte(expect), []byte(strings.ToLower(sign))) {
		a.fail(c, "签名验证失败")
		return
	}

	//nonce 去重，同一窗口内不允许重复
	if a.NonceCheck {
		ok, msg := a.checkNonce(c, appKey, nonce)
		if !ok {
			a.fail(c, msg)
			return
		}
	}

	//校验接口访问范围
	if len(app.Apis) > 0 && !misc.IsItemLike(app.Apis, c.Request.URL.Path) {
		a.fail(c, "无权访问该接口")
		return
	}

	//存储在请求中
	c.Set("curApp", app)
}

// 生成签名，Go 侧调用第三方接口时可直接复用
func Sign(appSecret, method, path string, query url.Values, body []byte, timestamp, nonce string) string {
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write([]byte(BuildCanonical(method, path, query, body, timestamp, nonce)))
	return hex.EncodeToString(mac.Sum(nil))
}

// 拼接待签名的规范化串，各语言实现必须与此完全一致
func BuildCanonical(method, path string, query url.Values, body []byte, timestamp, nonce string) string {
	bodyHash := sha256.Sum256(body)
	return strings.Join([]string{
		strings.ToUpper(method),
		path,
		CanonicalQuery(query),
		hex.EncodeToString(bodyHash[:]),
		timestamp,
		nonce,
	}, "\n")
}

// 规范化 query，按 key 升序、同 key 的值升序，值使用解码后的原文（不做转义）
func CanonicalQuery(query url.Values) string {
	if len(query) == 0 {
		return ""
	}

	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(query))
	for _, k := range keys {
		vals := query[k]
		sort.Strings(vals)
		for _, v := range vals {
			parts = append(parts, k+"="+v)
		}
	}

	return strings.Join(parts, "&")
}

func (a *apiAuth) GetCurApp(c *gin.Context) misc.AppInfo {
	app, ok := c.Get("curApp")
	if !ok {
		return misc.AppInfo{}
	}
	return app.(misc.AppInfo)
}

func (a *apiAuth) fail(c *gin.Context, msg string) {
	br.Brs.Okc(c, a.FailCode, msg)
	c.Abort()
}

// 读取请求体并还原，保证后续 handler 还能正常读取
func (a *apiAuth) readBody(c *gin.Context) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}

	buf, err := io.ReadAll(io.LimitReader(c.Request.Body, a.MaxBody+1))
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(buf))

	return buf, nil
}

// nonce 去重，窗口内同一 ApiKey 的同一 nonce 只允许出现一次。
// 返回是否放行，以及拒绝时的原因
func (a *apiAuth) checkNonce(c *gin.Context, appKey, nonce string) (bool, string) {
	key := "auth:api:nonce:" + appKey + ":" + nonce
	// 存活时间略大于时间戳窗口，覆盖整个可重放区间
	ttl := a.Expire + time.Minute

	var ok bool
	var err error
	if conf.Configs.GetBool("is.redis.cluster") {
		rdc, exists := redis.Rdbc["default"]
		if !exists {
			return a.NonceFailOpen, "nonce 校验服务不可用"
		}
		ok, err = rdc.SetNX(c, key, 1, ttl).Result()
	} else {
		rdb, exists := redis.Rdb["default"]
		if !exists {
			return a.NonceFailOpen, "nonce 校验服务不可用"
		}
		ok, err = rdb.SetNX(c, key, 1, ttl).Result()
	}

	if err != nil {
		// Redis 不可用，按配置决定是放行还是拒绝
		return a.NonceFailOpen, "nonce 校验服务不可用"
	}

	if !ok {
		return false, "重复请求"
	}

	return true, ""
}

// 兼容秒与毫秒两种时间戳，毫秒值必然大于 1e12
func parseTimestamp(ts string) (time.Time, bool) {
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}, false
	}
	if n > 1e12 {
		return time.UnixMilli(n), true
	}
	return time.Unix(n, 0), true
}
