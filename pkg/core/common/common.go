package common

import (
	"net"
	"os"
	"strconv"
	"strings"
	logging "synapsor/pkg/core/log"
	"time"
)

var (
	DialTimeout = 5
)

// check grpc server
func CheckGRPCSerer(addr string) bool {
	// gRPC Dial Timeout
	gRPCDialTimeout := os.Getenv("GRPC_DIAL_TIMEOUT")
	gRPCDialTimeoutInt := DialTimeout
	if gRPCDialTimeout != "" {
		gRPCDialTimeoutInt, _ = strconv.Atoi(gRPCDialTimeout)
	}

	checkServiceType := os.Getenv("GRPC_DIAL_CHECK_SVC_TYPE")
	switch checkServiceType {
	case "svcName":
		// tcp dial check
		if !doCheckGRPCSerer(addr, gRPCDialTimeoutInt) {
			return false
		}
	case "svcIP":
		// tcp dial check
		hosts := strings.Split(addr, ":")
		serviceHost := hosts[0]
		servicePort := hosts[1]
		addrIP, err := net.ResolveIPAddr("ip", serviceHost)
		if err != nil {
			logging.ERROR.Error("Resolution error", err.Error())
		}
		PrintDebugLog(hosts, addrIP)
		if addrIP != nil {
			addr = addrIP.IP.String() + ":" + servicePort
			checkStatus := doCheckGRPCSerer(addr, gRPCDialTimeoutInt)
			PrintDebugLog(addr, checkStatus)
			return checkStatus
		}
		return false
	case "svcNameAndsvcIP":
		PrintDebugLog("check container use svcName and svcIP")
		// tcp dial check
		if !doCheckGRPCSerer(addr, gRPCDialTimeoutInt) {
			hosts := strings.Split(addr, ":")
			serviceHost := hosts[0]
			servicePort := hosts[1]
			addrIP, err := net.ResolveIPAddr("ip", serviceHost)
			if err != nil {
				logging.ERROR.Error("Resolution error", err.Error())
			}
			PrintDebugLog(hosts, addrIP)
			if addrIP != nil {
				addr = addrIP.IP.String() + ":" + servicePort
				checkStatus := doCheckGRPCSerer(addr, gRPCDialTimeoutInt)
				PrintDebugLog(addr, checkStatus)
				return checkStatus
			}
			return false
		}
	default:
		// tcp dial check
		if !doCheckGRPCSerer(addr, gRPCDialTimeoutInt) {
			return false
		}
	}

	return true
}

// do checkGRPCSerer
func doCheckGRPCSerer(addr string, gRPCDialTimeoutInt int) bool {
	// tcp dial check
	conn, err := net.DialTimeout("tcp", addr, time.Duration(gRPCDialTimeoutInt)*time.Second)
	if err != nil || conn == nil {
		return false
	}
	defer conn.Close()
	return true
}

// print debug log
func PrintDebugLog(logStr ...interface{}) {
	// print debug log
	vsDebug := os.Getenv("VS_DEBUG")
	if vsDebug == "true" {
		logging.DEBUG.Debug("print debug log: ", logStr)
	}
}
