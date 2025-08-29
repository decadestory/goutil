package misc

import (
	"fmt"
	"net"
	"strings"
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
