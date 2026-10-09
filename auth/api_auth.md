# 第三方接口认证（ApiKey / AppSecret）

面向第三方接入方：所有对外开放接口需在请求头携带签名，服务端用中间件校验。

## 一、请求头

| 请求头 | 说明 |
| --- | --- |
| `X-Api-Key` | 平台分配的第三方标识 |
| `X-Timestamp` | 请求时间戳，**秒或毫秒**均可（自动识别） |
| `X-Nonce` | 随机串，建议 UUID/随机十六进制，同一窗口内不可重复 |
| `X-Sign` | 签名值，hex 小写（大写也接受） |

`AppSecret` **不参与传输**，仅用于本地计算签名。

## 二、签名规则

### 2.1 规范化串

先拼出 6 行文本，用换行符 `\n` 连接：

```
METHOD\nPATH\nQUERY\nSHA256_HEX(BODY)\nTIMESTAMP\nNONCE
```

| 位置 | 取值 |
| --- | --- |
| `METHOD` | HTTP 方法，**大写**，如 `POST` |
| `PATH` | URL 路径，不含 query，如 `/v1/order` |
| `QUERY` | 见 2.2 |
| `SHA256_HEX(BODY)` | 请求体原始字节的 SHA-256，hex 小写；无请求体时为**空串**的哈希 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `TIMESTAMP` | 与请求头 `X-Timestamp` **完全一致**的原始字符串 |
| `NONCE` | 与请求头 `X-Nonce` 完全一致的原始字符串 |

### 2.2 QUERY 的规范化

- 按参数名**升序**排列；
- 同名多值的参数，值也**升序**排列，逐个输出为 `key=value`；
- 多组之间用 `&` 连接；
- **值使用 URL 解码后的原文，不做任何转义**（空格就是空格，不是 `%20` 也不是 `+`）；
- 无参数时该行为空串。

`?b=2&a=1` 与 `?a=1&b=2` 规范化后都是 `a=1&b=2`，因此**客户端拼接参数顺序不影响验签**。

### 2.3 计算签名

```
sign = hex_lower( HMAC_SHA256(key = AppSecret, data = 规范化串) )
```

### 2.4 完整示例

请求：

```
POST /v1/order?a=1&b=2 HTTP/1.1
X-Api-Key: ak_demo
X-Timestamp: 1700000000
X-Nonce: abc123

{"orderId":"123"}
```

规范化串（`\n` 处已换行）：

```
POST
/v1/order
a=1&b=2
379214f2718333da854418b2ced7435430af05f9082d793abd6b7cd9159cdb75
1700000000
abc123
```

AppSecret 为 `sk_test_secret` 时：

```
X-Sign: db519254e9d97b874f855138f6cd3c790f5054553a8d7b24b112500748d83fb9
```

该向量已用 Python 标准库与 openssl 双向核对，可直接作为各语言自测用例。

## 三、各语言实现

### Go

中间件同包内已提供实现，直接复用：

```go
import "github.com/decadestory/goutil/auth"

sign := auth.Sign(secret, "POST", "/v1/order", query, body, ts, nonce)
```

### Python

```python
import hashlib, hmac, time, uuid


def canonical_query(params):
    """params: dict[str, str 或 list[str]]"""
    parts = []
    for k in sorted(params):
        vals = params[k] if isinstance(params[k], list) else [params[k]]
        for v in sorted(vals):
            parts.append("%s=%s" % (k, v))
    return "&".join(parts)


def sign(secret, method, path, params, body, ts, nonce):
    canonical = "\n".join([
        method.upper(),
        path,
        canonical_query(params),
        hashlib.sha256(body).hexdigest(),
        ts,
        nonce,
    ])
    return hmac.new(secret.encode(), canonical.encode(), hashlib.sha256).hexdigest()


ts = str(int(time.time()))
nonce = uuid.uuid4().hex
body = b'{"orderId":"123"}'
sign_value = sign("sk_test_secret", "POST", "/v1/order", {"a": "1", "b": "2"}, body, ts, nonce)
```

### JavaScript / Node.js

```js
const crypto = require('crypto');

function canonicalQuery(params) {
  const out = [];
  for (const k of Object.keys(params).sort()) {
    const vals = Array.isArray(params[k]) ? params[k] : [params[k]];
    for (const v of vals.slice().sort()) out.push(`${k}=${v}`);
  }
  return out.join('&');
}

function sign(secret, method, path, params, body, ts, nonce) {
  const bodyHash = crypto.createHash('sha256').update(body).digest('hex');
  const canonical = [method.toUpperCase(), path, canonicalQuery(params), bodyHash, ts, nonce].join('\n');
  return crypto.createHmac('sha256', secret).update(canonical).digest('hex');
}
```

浏览器端用 `crypto.subtle` 时注意：`HMAC` 的 key 用 `TextEncoder` 编码，`SHA-256` 对空 body 也要传空 `Uint8Array` 而不是 `undefined`。

### Java

```java
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.*;

public class ApiSign {

    private static String hex(byte[] bytes) {
        StringBuilder sb = new StringBuilder(bytes.length * 2);
        for (byte b : bytes) {
            sb.append(String.format("%02x", b));
        }
        return sb.toString();
    }

    private static String canonicalQuery(Map<String, List<String>> params) {
        List<String> keys = new ArrayList<>(params.keySet());
        Collections.sort(keys);

        List<String> parts = new ArrayList<>();
        for (String k : keys) {
            List<String> vals = new ArrayList<>(params.get(k));
            Collections.sort(vals);
            for (String v : vals) {
                parts.add(k + "=" + v);
            }
        }
        return String.join("&", parts);
    }

    public static String sign(String secret, String method, String path,
                             Map<String, List<String>> params, byte[] body,
                             String ts, String nonce) throws Exception {
        String bodyHash = hex(MessageDigest.getInstance("SHA-256").digest(body));
        String canonical = String.join("\n",
                method.toUpperCase(), path, canonicalQuery(params), bodyHash, ts, nonce);

        Mac mac = Mac.getInstance("HmacSHA256");
        mac.init(new SecretKeySpec(secret.getBytes(StandardCharsets.UTF_8), "HmacSHA256"));
        return hex(mac.doFinal(canonical.getBytes(StandardCharsets.UTF_8)));
    }
}
```

### PHP

```php
<?php
function canonical_query(array $params): string {
    ksort($params);
    $parts = [];
    foreach ($params as $k => $v) {
        $vals = is_array($v) ? $v : [$v];
        sort($vals);
        foreach ($vals as $each) {
            $parts[] = $k . '=' . $each;
        }
    }
    return implode('&', $parts);
}

function api_sign(string $secret, string $method, string $path,
                  array $params, string $body, string $ts, string $nonce): string {
    $canonical = implode("\n", [
        strtoupper($method),
        $path,
        canonical_query($params),
        hash('sha256', $body),
        $ts,
        $nonce,
    ]);
    return hash_hmac('sha256', $canonical, $secret);
}
```

## 四、服务端配置

```toml
[auth.api]
# 不校验签名的路径
ignore = ["/health", "/ui", "/swagger"]
# 时间戳有效窗口（分钟）
expire = 5
# 参与签名的请求体上限（字节），超过直接拒绝
maxbody = 4194304
# 请求头名称，一般无需修改
keyname = "X-Api-Key"
timestampname = "X-Timestamp"
noncename = "X-Nonce"
signname = "X-Sign"
# 认证失败返回的业务码
failcode = 401
# 是否校验 nonce 防重放（依赖 Redis）
noncecheck = true
# Redis 不可用时是否放行。false=拒绝，避免故障期间被重放
noncefailopen = false
```

### 4.1 密钥来源

第三方应用**不再从配置文件读取**，必须给 `AuthApis.AppFunc` 赋值，中间件每次请求都通过它按 ApiKey 取应用信息：

```go
auth.AuthApis.AppFunc = func(c *gin.Context, appKey string) (misc.AppInfo, bool) {
    app, err := appRepo.GetByKey(c, appKey)
    if err != nil {
        return misc.AppInfo{}, false
    }
    return app, true
}
```

未赋值时中间件会在请求到达时 panic，务必在注册路由前设置。

返回值 `misc.AppInfo` 中与认证相关的字段：

| 字段 | 说明 |
| --- | --- |
| `Key` | ApiKey，与请求头一致（回调内部查库用） |
| `Secret` | AppSecret，参与验签 |
| `Valid` | 为 `false` 时视为无效的 ApiKey |
| `Apis` | 允许访问的接口，为空表示不限制 |

### 4.2 挂载

```go
r := gin.New()
r.Use(exception.Errors.Recover)
r.Use(auth.AuthApis.AuthMiddleware)
```

业务侧取当前第三方：

```go
app := auth.AuthApis.GetCurApp(c)
```

## 五、返回格式

沿用 `br` 统一返回体，**HTTP 状态码始终为 200**，以响应体中的 `code` 判断结果：

```json
{ "code": 401, "data": null, "msg": "签名验证失败" }
```

`code == 1` 表示成功。失败 `msg` 一览：

| msg | 原因 |
| --- | --- |
| 缺少认证参数 | 四个认证头有缺失 |
| 时间戳格式错误 | `X-Timestamp` 不是整数 |
| 请求已过期 | 超出 `expire` 窗口 |
| 无效的 ApiKey | `AppFunc` 未查到该 ApiKey，或 `Valid` 为 `false` |
| 请求体读取失败 | 读取 body 出错 |
| 请求体过大 | 超过 `maxbody` |
| 签名验证失败 | 签名不匹配，或密钥错误 |
| 重复请求 | nonce 已被使用 |
| nonce 校验服务不可用 | Redis 故障且 `noncefailopen = false` |
| 无权访问该接口 | 超出该第三方授权的 `apis` 范围 |

## 六、常见坑

1. **query 不要二次转义**。规则要求解码后的原文，`?q=%E4%B8%AD` 应参与签名的值是 `中`。若用现成的 URL 编码函数拼串，注意空格：`%20` 与 `+` 解码后都是空格，三种写法结果一致。
2. **无参数时不要拼 `?`**。`/v1/order` 与 `/v1/order?` 的 `PATH` 相同、`QUERY` 都为空串，但别让中间件把空 query 当成 `"?"`。
3. **body 用原始字节**，不要重新序列化 JSON 后再算哈希（键顺序、空格、转义都会变）。
4. **GET 的 body** 传空字节数组，不要传 `null`。
5. **时间戳、nonce 头与参与签名的值必须逐字符一致**，不要做 trim 或格式化。
6. **每个请求换新 nonce**，重试也要换，否则会被判定为重放。
7. **`Content-Type` 不参与签名**，服务端不校验它。
8. 服务端会读取并还原请求体，业务 handler 仍可正常 `c.ShouldBindJSON`。
