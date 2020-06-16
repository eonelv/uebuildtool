// NetUtils
package utils

import (
	. "core"
	"errors"
	"fmt"
	"net"
	"strings"
)

var AllowMacAddress = map[string]uint8{
	"0C-C4-7A-6E-8D-D2": 1, //192.168.1.19
	"A8-5E-45-30-ED-1A": 1, //LV
}

func GetLocalIP() (ipv4 string, err error) {
	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet // IP地址
		isIpNet bool
	)
	// 获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 取第一个非lo的网卡IP
	for _, addr = range addrs {
		// 这个网络地址是IP地址: ipv4, ipv6

		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过IPV6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String() // 192.168.1.1

				LogInfo(ipv4)
				if strings.HasPrefix(ipv4, "192.168.") {
					return
				}
			}
		}
	}

	err = errors.New("No IP")
	return
}

func GetMacAddrs() (macAddrs []string) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		LogError(fmt.Printf("fail to get net interfaces: %v", err))
		return macAddrs
	}

	for _, netInterface := range netInterfaces {
		macAddr := netInterface.HardwareAddr.String()
		if len(macAddr) == 0 {
			continue
		}

		macAddrs = append(macAddrs, macAddr)
	}
	return macAddrs
}
