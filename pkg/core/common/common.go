package common

import (
	"fmt"
	"math"
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

// message
type Message struct {
	Code    int
	Err     error
	Message string
	Data    interface{}
}

//send message internal
func SendMessageIn(code int, err error) Message {
	return Message{
		Code:    code,
		Err:     err,
		Message: "",
		Data:    nil,
	}
}

//get ccnum div by radio
func GetCCNumDivByRatio(engineCCSumStr, engine2CoreRatio string) (string, error) {
	//setting engine ratio
	engineCCSumFloat, errECCSumTrans := strconv.ParseFloat(engineCCSumStr, 64)
	if errECCSumTrans != nil {
		return "", errECCSumTrans
	}
	engine2CoreRatioFloat, errERadioTrans := strconv.ParseFloat(engine2CoreRatio, 64)
	if errERadioTrans != nil {
		return "", errERadioTrans
	}
	ccnumResult := engineCCSumFloat / engine2CoreRatioFloat
	ccnumResultStr := fmt.Sprintf("%.0f", math.Ceil(ccnumResult))
	return ccnumResultStr, nil
}

//get ccnum div by radio
func GetCCNumTimesByRatio(engineCCSumStr, engine2CoreRatio string, divNum float64) (string, error) {
	//setting engine ratio
	engineCCSumFloat, errECCSumTrans := strconv.ParseFloat(engineCCSumStr, 64)
	if errECCSumTrans != nil {
		return "", errECCSumTrans
	}
	engine2CoreRatioFloat, errERadioTrans := strconv.ParseFloat(engine2CoreRatio, 64)
	if errERadioTrans != nil {
		return "", errERadioTrans
	}
	ccnumResult := engineCCSumFloat * 1.5 * engine2CoreRatioFloat / divNum
	if engine2CoreRatioFloat/divNum > 5 {
		ccnumResult = engineCCSumFloat * 1.1 * engine2CoreRatioFloat / divNum
	}
	ccnumResultStr := fmt.Sprintf("%.0f", math.Ceil(ccnumResult))
	return ccnumResultStr, nil
}

//get str by number
func GetNumByStr(numStr, operator string, operatorNum int) string {
	//setting engine ratio
	num, errNumTrans := strconv.ParseFloat(numStr, 64)
	if errNumTrans != nil {
		return numStr
	}
	var numResult float64
	switch operator { //plus 加 minus 减 times 乘 divded 除
	case "plus":
		numResult = num + float64(operatorNum)
	case "minus":
		numResult = num - float64(operatorNum)
	case "times":
		numResult = num * float64(operatorNum)
	case "divided":
		numResult = num / float64(operatorNum)
	}
	numResultStr := fmt.Sprintf("%.0f", math.Ceil(numResult))
	return numResultStr
}

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
