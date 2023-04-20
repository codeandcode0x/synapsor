package proxy

import (
	"fmt"
	"net"
	"strings"
	logging "synapsor/pkg/core/log"
	grpcPool "synapsor/pkg/plugins/pool/grpc"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var grpcViper *viper.Viper

func init() {
	// get config
	grpcViper = viper.New()
	grpcViper.SetConfigName("ProxyConfig")
	grpcViper.SetConfigType("yaml")
	grpcViper.AddConfigPath("config/")
	err := grpcViper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error get pool config file: %s", err))
	}
}

// rpc server
func (plugin *Plugin) GRPCServer() {
	// init config
	rvRoot := grpcViper.AllSettings()["proxy"]
	setting := rvRoot.(map[string]interface{})["setting"].(map[string]interface{})
	// init grpc server
	proxys := rvRoot.(map[string]interface{})["proxy_list"]
	for _, v := range proxys.([]interface{}) {
		vMap := v.(map[string]interface{})
		// get pool enabl
		if !vMap["POOL_ENABLED"].(bool) {
			continue
		}
		// run grpc server
		go plugin.vsGrpcInitServer(setting[strings.ToLower("LISTEN_PROXY_ADDR")].(string)+":"+vMap["PROXY_PORT"].(string), vMap["PROXY_NAME"].(string))
	}
}

//grpc server
func (plugin Plugin) vsGrpcInitServer(addr, serviceName string) {
	logging.Log.Info(serviceName, " gRPC Server start ...")
	//goroutine break
	defer func() {
		plugin.Status <- false
	}()

	// get gRPC port
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logging.ERROR.Errorf("failed to listen: %v", err)
		return
	}
	// grpc new server
	srv := grpc.NewServer(grpc.CustomCodec(grpcPool.Codec()),
		grpc.UnknownServiceHandler(grpcPool.TransparentHandler(grpcPool.GrpcProxyTransport)))
	// register service
	grpcPool.RegisterService(srv, grpcPool.GrpcProxyTransport,
		"PingEmpty",
		"Ping",
		"PingError",
		"PingList",
	)
	reflection.Register(srv)
	// start ser listen
	err = srv.Serve(lis)
	if err != nil {
		logging.ERROR.Errorf("failed to serve: %v", err)
		return
	}
}
