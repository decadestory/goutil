package auth

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/decadestory/goutil/misc"
	"github.com/gin-gonic/gin"
)

const (
	testAppKey    = "ak_test"
	testAppSecret = "sk_test_secret"
)

func init() {
	gin.SetMode(gin.TestMode)

	AuthApis.Apps = func(c *gin.Context, appKey string) (misc.AppInfo, bool) {
		if appKey == testAppKey {
			return misc.AppInfo{Key: testAppKey, Secret: testAppSecret, Name: "测试第三方", Valid: true}, true
		}
		return misc.AppInfo{}, false
	}
	// 测试环境不接 Redis，nonce 校验走 fail-open 分支
	AuthApis.NonceFailOpen = true
}

func newCtx(method, target string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, bytes.NewReader(body))
	return c, w
}

// 用给定密钥和请求内容签名并带上认证头
func signRequest(c *gin.Context, secret string, body []byte, ts, nonce string) {
	sign := Sign(secret, c.Request.Method, c.Request.URL.EscapedPath(),
		c.Request.URL.Query(), body, ts, nonce)

	c.Request.Header.Set(AuthApis.KeyName, testAppKey)
	c.Request.Header.Set(AuthApis.TimestampName, ts)
	c.Request.Header.Set(AuthApis.NonceName, nonce)
	c.Request.Header.Set(AuthApis.SignName, sign)
}

func nowTS() string { return strconv.FormatInt(time.Now().Unix(), 10) }

// 跨语言一致性：该向量由 Python(hmac) 与 openssl 独立计算并核对一致
func TestSignCrossLanguageVector(t *testing.T) {
	got := Sign(
		"sk_test_secret",
		"POST",
		"/v1/order",
		map[string][]string{"a": {"1"}, "b": {"2"}},
		[]byte(`{"orderId":"123"}`),
		"1700000000",
		"abc123",
	)

	const want = "db519254e9d97b874f855138f6cd3c790f5054553a8d7b24b112500748d83fb9"
	if got != want {
		t.Fatalf("签名与跨语言参考实现不一致\n got: %s\nwant: %s", got, want)
	}
}

func TestCanonicalQuerySorted(t *testing.T) {
	query := map[string][]string{"b": {"2"}, "a": {"1", "0"}}
	if got := CanonicalQuery(query); got != "a=0&a=1&b=2" {
		t.Fatalf("query 规范化顺序错误: %s", got)
	}

	if got := CanonicalQuery(nil); got != "" {
		t.Fatalf("空 query 应为空串, 实际: %q", got)
	}
}

func TestAuthApiPass(t *testing.T) {
	body := []byte(`{"orderId":"123"}`)
	c, w := newCtx(http.MethodPost, "/v1/order?b=2&a=1", body)
	signRequest(c, testAppSecret, body, nowTS(), "nonce-1")

	AuthApis.AuthMiddleware(c)

	if c.IsAborted() {
		t.Fatalf("合法请求被拒绝: %s", w.Body.String())
	}

	app := AuthApis.GetCurApp(c)
	if app.Key != testAppKey || app.Name != "测试第三方" {
		t.Fatalf("curApp 未正确注入: %+v", app)
	}

	// 请求体必须还原，业务侧还要继续读
	got, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("读取还原后的 body 失败: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("body 未还原\n got: %s\nwant: %s", got, body)
	}
}

func TestAuthApiQueryOrderNotMatter(t *testing.T) {
	body := []byte(`{}`)
	ts := nowTS()
	// 客户端按 a=1&b=2 签名，实际请求参数顺序相反，仍需通过
	c, w := newCtx(http.MethodGet, "/v1/order?b=2&a=1", body)
	sign := Sign(testAppSecret, "GET", "/v1/order",
		map[string][]string{"a": {"1"}, "b": {"2"}}, body, ts, "nonce-q")

	c.Request.Header.Set(AuthApis.KeyName, testAppKey)
	c.Request.Header.Set(AuthApis.TimestampName, ts)
	c.Request.Header.Set(AuthApis.NonceName, "nonce-q")
	c.Request.Header.Set(AuthApis.SignName, sign)

	AuthApis.AuthMiddleware(c)
	if c.IsAborted() {
		t.Fatalf("query 参数顺序不同导致验签失败: %s", w.Body.String())
	}
}

func TestAuthApiMillisTimestamp(t *testing.T) {
	body := []byte(`{}`)
	c, w := newCtx(http.MethodGet, "/v1/ping", body)
	signRequest(c, testAppSecret, body, strconv.FormatInt(time.Now().UnixMilli(), 10), "nonce-ms")

	AuthApis.AuthMiddleware(c)
	if c.IsAborted() {
		t.Fatalf("毫秒时间戳未被兼容: %s", w.Body.String())
	}
}

func TestAuthApiWrongSecret(t *testing.T) {
	body := []byte(`{}`)
	c, _ := newCtx(http.MethodGet, "/v1/order", body)
	signRequest(c, "sk_wrong_secret", body, nowTS(), "nonce-2")

	AuthApis.AuthMiddleware(c)

	if !c.IsAborted() {
		t.Fatal("错误密钥的请求应当被拒绝")
	}
}

func TestAuthApiTamperedBody(t *testing.T) {
	original := []byte(`{"amount":1}`)
	tampered := []byte(`{"amount":9999}`)

	c, _ := newCtx(http.MethodPost, "/v1/order", tampered)
	// 用原始 body 签名，实际发送被篡改的 body
	signRequest(c, testAppSecret, original, nowTS(), "nonce-3")

	AuthApis.AuthMiddleware(c)

	if !c.IsAborted() {
		t.Fatal("篡改请求体后签名应当校验失败")
	}
}

func TestAuthApiExpiredTimestamp(t *testing.T) {
	body := []byte(`{}`)
	c, _ := newCtx(http.MethodGet, "/v1/order", body)
	old := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	signRequest(c, testAppSecret, body, old, "nonce-4")

	AuthApis.AuthMiddleware(c)

	if !c.IsAborted() {
		t.Fatal("超出时间窗口的请求应当被拒绝")
	}
}

func TestAuthApiMissingHeader(t *testing.T) {
	body := []byte(`{}`)
	c, _ := newCtx(http.MethodGet, "/v1/order", body)
	// 不带任何认证头
	AuthApis.AuthMiddleware(c)

	if !c.IsAborted() {
		t.Fatal("缺少认证头的请求应当被拒绝")
	}
}

func TestAuthApiUnknownKey(t *testing.T) {
	body := []byte(`{}`)
	c, _ := newCtx(http.MethodGet, "/v1/order", body)
	signRequest(c, testAppSecret, body, nowTS(), "nonce-5")
	c.Request.Header.Set(AuthApis.KeyName, "ak_not_exists")

	AuthApis.AuthMiddleware(c)

	if !c.IsAborted() {
		t.Fatal("未注册的 ApiKey 应当被拒绝")
	}
}

func TestAuthApiIgnorePath(t *testing.T) {
	body := []byte(`{}`)
	c, _ := newCtx(http.MethodGet, "/health", body)

	AuthApis.AuthMiddleware(c)

	if c.IsAborted() {
		t.Fatal("白名单接口不应校验签名")
	}
}
