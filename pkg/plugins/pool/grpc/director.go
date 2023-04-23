package grpc

import (
	"context"
	"math/rand"
	"os"
	"strings"
	logging "synapsor/pkg/core/log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// options struct
type Options struct {
	Dial                 func(address string) (*grpc.ClientConn, error)
	PoolModel            int    // Pool 模型
	MaxIdle              int    // 最大空闲数量
	MaxActive            int    // 最大活跃连接数
	MaxConcurrentStreams int    // 最大并发数量
	Reusable             bool   // 是否可以复用
	RequestIdleTime      int    // request idle 时间
	RequestMaxLife       int    // request max life 时间
	RequestTimeOut       int    // request timeout
	GatewayProxyAddr     string // grpc gateway 代理地址
	GatewayProxyPort     string // grpc gateway 代理端口
	PoolStatus           bool   // 连接池是否开启
}

var DEFAULT_PROXY = "default"

func init() {
	rand.Seed(time.Now().Unix())
}

// grpc proxy director
func GrpcProxyTransport(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, *Client, error) {
	if strings.HasPrefix(fullMethodName, "/ivc.v1.internal") {
		return nil, nil, nil, status.Errorf(codes.Unimplemented, "invaild or unsupported method")
	}
	var err error
	var conn *Client
	// setting md data
	md, mdExists := metadata.FromIncomingContext(ctx)
	outCtx, _ := context.WithCancel(ctx)
	outCtx = metadata.NewOutgoingContext(outCtx, md.Copy())

	if defaultProxy {
		conn, err = balancePool(connPools[DEFAULT_PROXY], connProxy[DEFAULT_PROXY]["proxyModel"].(string)).Acquire(ctx)
	} else if mdExists {
		proxyName, proxyNameExists := md["proxy"]
		if !proxyNameExists {
			return nil, nil, nil, status.Errorf(codes.Unimplemented, "proxy name not exist")
		}
		conn, err = balancePool(connPools[proxyName[0]], connProxy[proxyName[0]]["proxyModel"].(string)).Acquire(ctx)
	}
	// conn not nil
	if conn != nil {
		return outCtx, conn.ClientConn, conn, err
	}

	// return unknow error
	return nil, nil, nil, status.Errorf(codes.Unimplemented, "unknown method")
}

// gRPC proxy
func balancePool(pools map[string]*Pool, proxyModel string) *Pool {
	// var sumSize, size int
	var index string
	if len(pools) < 1 {
		return nil
	}

	switch strings.ToLower(proxyModel) {
	case "randomweight":
		index = randomWeightBalance(pools)
	case "minconn":
		index = minConnBalance(pools)
	default:
		index = randomWeightBalance(pools)
	}
	vsDebug := os.Getenv("VS_DEBUG")
	if vsDebug == "true" {
		logging.DEBUG.Debug("print debug log grpc balance index: ", index, proxyModel)
	}
	pools[index].sumRequestTimes += 1
	return pools[index]
}

// get gRPC min conn
func minConnBalance(pools map[string]*Pool) string {
	var sumSize, size int
	var index string
	var indexRand []string
	// 最小连接数
	for k, pool := range pools {
		if !pool.status {
			continue
		}
		pSize := pool.Size()
		sumSize += int(pool.Size())
		if size <= pSize {
			size = int(pSize)
			index = k
		}
		indexRand = append(indexRand, k)
	}
	// 随机
	if sumSize == 0 {
		index = indexRand[rand.Intn(len(indexRand))]
	}
	return index
}

// random with weight proxy
func weightedRandomIndex(weights []float32) int {
	if len(weights) == 1 {
		return 0
	}
	var sum float32 = 0.0
	for _, w := range weights {
		sum += w
	}
	r := rand.Float32() * sum
	var t float32 = 0.0
	for i, w := range weights {
		t += w
		if t > r {
			return i
		}
	}
	return len(weights) - 1
}

// random weight balance
func randomWeightBalance(pools map[string]*Pool) string {
	var weights = []float32{}
	var indexRand []string
	for k, pool := range pools {
		weight := pool.capacity
		weights = append(weights, float32(weight))
		indexRand = append(indexRand, k)
	}

	poolIndex := indexRand[weightedRandomIndex(weights)]
	vsDebug := os.Getenv("VS_DEBUG")
	if vsDebug == "true" {
		logging.DEBUG.Debug("print debug log grpc balance randomWeightBalance: ", weights, indexRand, poolIndex)
	}
	return poolIndex
}
