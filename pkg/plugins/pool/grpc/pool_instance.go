package grpc

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"synapsor/pkg/core/common"
	logging "synapsor/pkg/core/log"
	"synapsor/pkg/plugins"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// plugin
type Plugin plugins.Plugin

// config viper
var routerViper *viper.Viper

// conn pools
var connPools = make(map[string]map[string]*Pool)

// conn proxy
var connProxy = make(map[string]map[string]interface{})

// failed pool status
var failedPoolStatus bool = false

// whether use default proxy
var defaultProxy bool = false

// const
var (
	DialTimeout                 = 5 * time.Second
	BackoffMaxDelay             = 3 * time.Second
	KeepAliveTime               = time.Duration(3) * time.Second
	KeepAliveTimeout            = time.Duration(3) * time.Second
	InitialWindowSize     int32 = 1 << 30
	InitialConnWindowSize int32 = 1 << 30 // 1073741824
	MaxSendMsgSize              = 4 << 30 // 4294967296
	MaxRecvMsgSize              = 4 << 30
	GatewayProxyAddr            = ""
	NetWorkMode                 = 1
)

// init grpc pool
func InitGrpcConnPool() {
	// get config
	routerViper = viper.New()
	routerViper.SetConfigName("ProxyConfig")
	routerViper.SetConfigType("yaml")
	routerViper.AddConfigPath("config/")
	err := routerViper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error get config file: %s", err))
	}
	// proxyConfig := routerViper.AllSettings()["proxy"].([]interface{})
	rvRoot := routerViper.AllSettings()["proxy"]
	proxyConfig := rvRoot.(map[string]interface{})["proxy_list"].([]interface{})
	for _, v := range proxyConfig {
		proxyMap := v.(map[string]interface{})
		poolEnabled := proxyMap["ENABLED"].(bool)
		if !poolEnabled {
			continue
		}

		requestIdleTime := proxyMap["REQUEST_IDLE_TIME"].(int)
		requestMaxLife := proxyMap["REQUEST_MAX_LIFE"].(int)
		requestTimeout := proxyMap["REQUEST_TIMEOUT"].(int)
		defaultGrpcConnNum := proxyMap["DEFAULT_GRPC_CONN_NUM"].(int)
		grpcRequestReusable := proxyMap["GRPC_REQUEST_REUSABLE"].(bool)
		poolModel := proxyMap["POOL_MODEL"].(int)
		proxyName := proxyMap["PROXY_NAME"].(string)
		proxyModel := proxyMap["PROXY_MODEL"].(string)
		// setting default proxy
		if len(proxyConfig) == 1 && proxyName == "default" {
			defaultProxy = true
		}
		// proxy map loop
		for _, endPoint := range proxyMap["GRPC_PROXY_ENDPOINTS"].([]interface{}) {
			endPointStr := endPoint.(string)
			endPointList := strings.Split(endPointStr, "#")
			if len(endPointList) < 2 {
				logging.ERROR.Error("init grpc connection error, config env invaild...")
				break
			}

			poolInitMap := map[string]interface{}{
				"grpcRequestReusable": grpcRequestReusable,
				"requestIdleTime":     requestIdleTime,
				"requestMaxLife":      requestMaxLife,
				"requestTimeout":      requestTimeout,
				"serverHost":          endPointList[0],
				"gatewayProxyPort":    endPointList[0],
				"poolEnabled":         poolEnabled,
			}

			poolInitMap["connNum"] = defaultGrpcConnNum
			poolInitMap["serverName"] = endPointList[0]
			poolInitMap["serverHost"] = endPointList[0]
			poolInitMap["proxyName"] = proxyName
			poolInitMap["proxyWeight"] = endPointList[1]
			poolInitMap["serviceCode"] = common.GenXid()
			poolInitMap["poolModel"] = poolModel
			poolInitMap["proxyModel"] = proxyModel

			initGrpcProxyPool(poolInitMap)

			logging.DEBUG.Debug("init grpc connection ", poolInitMap["serverName"], " finish ...")
		}
	}
}

// new grpc pool
func newGrpcPool(address string, option Options) (*Pool, error) {
	dial := func() (*grpc.ClientConn, error) {
		return grpcDial(address)
	}

	gp, err := NewPool(
		dial,
		int32(option.MaxIdle),
		int32(option.MaxActive),
		time.Duration(option.RequestIdleTime)*time.Second,
		time.Duration(option.RequestMaxLife)*time.Second,
		time.Duration(option.RequestTimeOut)*time.Second,
		option.PoolModel,
	)
	return gp, err
}

// release grpc pool
func ReleaseGrpcPool(proxyName, poolName string) {
	if _, ok := connPools[proxyName][poolName]; ok {
		connPools[proxyName][poolName].Close()
		delete(connPools[proxyName], poolName)
		if len(connPools[proxyName]) < 1 {
			delete(connPools, proxyName)
		}
	}
	logging.Log.Info("delete ", " pool ", poolName, " success")
}

// init grpc pool
func initGrpcProxyPool(data map[string]interface{}) {
	proxyName := data["proxyName"].(string)
	// get concur
	serverAddr := data["serverHost"].(string)
	connNum := data["connNum"].(int)
	// option setting
	op := Options{
		Dial:                 grpcDial,
		PoolModel:            data["poolModel"].(int),
		MaxIdle:              connNum,
		MaxActive:            connNum,
		MaxConcurrentStreams: 0,
		Reusable:             data["grpcRequestReusable"].(bool),
		RequestIdleTime:      data["requestIdleTime"].(int),
		RequestMaxLife:       data["requestMaxLife"].(int),
		RequestTimeOut:       data["requestTimeout"].(int),
		GatewayProxyAddr:     data["serverHost"].(string),
		GatewayProxyPort:     data["gatewayProxyPort"].(string),
		PoolStatus:           data["poolEnabled"].(bool),
	}
	// create pool
	pool, err := newGrpcPool(serverAddr, op)
	if err != nil {
		logging.ERROR.Error("failed to new pool: %v", err)
	}
	// setting pool remote addr
	pool.poolRemoteAddr = serverAddr
	pool.name = data["serviceCode"].(string)
	pool.code = proxyName
	weight, _ := strconv.Atoi(data["proxyWeight"].(string))
	pool.weight = int32(weight)
	pool.proxyModel = data["proxyModel"].(string)
	//init pool
	if _, ok := connPools[proxyName]; !ok {
		connPools[proxyName] = make(map[string]*Pool)
		connProxy[proxyName] = make(map[string]interface{})
	}
	connPools[proxyName][pool.name] = pool
	connProxy[proxyName]["proxyModel"] = data["proxyModel"]
}

// grpc dial
func grpcDial(address string) (*grpc.ClientConn, error) {
	ctx, ctxCancel := context.WithTimeout(context.Background(), DialTimeout)
	defer ctxCancel()
	gcc, err := grpc.DialContext(ctx, address,
		grpc.WithCodec(Codec()),
		grpc.WithInsecure(),
		grpc.WithBackoffMaxDelay(BackoffMaxDelay),
		grpc.WithInitialWindowSize(InitialWindowSize),
		grpc.WithInitialConnWindowSize(InitialConnWindowSize),
		grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(MaxSendMsgSize)),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxRecvMsgSize)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                3,
			Timeout:             3,
			PermitWithoutStream: true,
		}),
	)

	if err != nil {
		logging.ERROR.Error("grpc dial failed !", err)
	}

	return gcc, err
}

// get conn pool metrics data
func GetConnPoolMetricsData() map[string]map[string]int {
	connDataMap := make(map[string]map[string]int)
	for k, pools := range connPools {
		connDataMap[k] = make(map[string]int)
		for poolName, pool := range pools {
			connCurrent := pool.GetConnCurrent()
			connDataMap[k][poolName] = int(connCurrent)
		}
	}
	return connDataMap
}

// check grpc server task
func checkGRPCSererHealthTask() {
	for {
		// pool check
		for proxyName, poolMap := range connPools {
			for serviceCode, pool := range poolMap {
				checkStatus := common.CheckGRPCSerer(pool.poolRemoteAddr)
				if !checkStatus {
					for i := 0; i < 5; i++ {
						time.Sleep(1 * time.Second)
						checkStatus = common.CheckGRPCSerer(pool.poolRemoteAddr)
						if !checkStatus {
							continue
						} else {
							break
						}
					}
				}
				//check status
				if !checkStatus {
					logging.ERROR.Error("gRPC Server is down !")
					pool.status = false
					//exist failed pool
					failedPoolStatus = true
					ReleaseGrpcPool(proxyName, serviceCode)
				} else {
					//logging.Log.Info("gRPC Server is up !")
					pool.status = true
					//failed pool is up
					failedPoolStatus = false
				}
			}
		}
		time.Sleep(time.Second)
	} //end for
}

//encode (support base64)
func strEncode(str []byte) []byte {
	return []byte(base64.StdEncoding.EncodeToString(str))
}

//decode (support base64)
func strDecode(str []byte) ([]byte, error) {
	return base64.StdEncoding.DecodeString(string(str))
}
