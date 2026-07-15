package misc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/LinkinStars/go-scaffold/pkg/cryptor"
)

func GetIp() string {
	// 获取本机所有网络接口信息
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	// 遍历接口信息，找到非回环接口的 IPv4 地址
	for _, iface := range interfaces {
		// 排除回环接口和无效接口
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp != 0 {
			// 获取接口的 IP 地址
			addrs, err := iface.Addrs()
			if err != nil {
				fmt.Println("Failed to get IP addresses:", err)
				continue
			}

			// 遍历 IP 地址，找到 IPv4 地址
			for _, addr := range addrs {
				ip, ok := addr.(*net.IPNet)
				if ok && !ip.IP.IsLoopback() && ip.IP.To4() != nil {
					return ip.IP.String()
				}
			}
		}
	}

	return ""
}

// 泛型三元表达式函数
func Ternary[T any](cond bool, trueVal, falseVal T) T {
	if cond {
		return trueVal
	}
	return falseVal
}

func IsItemLike(items []string, item string) bool {
	for _, eachItem := range items {
		if strings.Contains(item, eachItem) {
			return true
		}
	}

	return false
}

// 用于前端加密解密
//
//	function aesEncrypt(data) {
//		var key = 'd8eab717abeca26cf5d0af2e216fa9f4'; // 16进制字符串
//		if (key.length > 32) {
//			key = key.slice(0, 32);
//		}
//		var cypherKey = CryptoJS.enc.Utf8.parse(key);
//		CryptoJS.pad.ZeroPadding.pad(cypherKey, 4);
//			var iv = CryptoJS.SHA256(key).toString();
//			var cfg = { iv: CryptoJS.enc.Utf8.parse(iv) };
//			return CryptoJS.AES.encrypt(data, cypherKey, cfg).toString();
//		}
//
// console.log(aesEncrypt('123456'));
func Decrypt(str string) string {

	res := ""
	defer func() {
		if err := recover(); err != nil {
			str = ""
		}
	}()

	key := "d8eab717abeca26cf5d0af2e216fa9f4"
	res = cryptor.AesSimpleDecrypt(str, key)
	return res
}

// 通用对称加密方案(AES-256-GCM)
func AesEncrypt(plainText string) (string, error) {
	key := []byte(KEY)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	iv := make([]byte, gcm.NonceSize()) // 12字节
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	// Seal 会返回:密文+Tag,并把结果append到dst(这里dst=iv,所以直接拼在iv后面)
	cipherData := gcm.Seal(iv, iv, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherData), nil
}

// 通用对称加密方案(AES-256-GCM)
func AesDecrypt(base64Data string) (string, error) {
	key := []byte(KEY)
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	ivLen := gcm.NonceSize()
	iv, cipherText := data[:ivLen], data[ivLen:]

	plainText, err := gcm.Open(nil, iv, cipherText, nil)
	if err != nil {
		return "", err
	}
	return string(plainText), nil
}
