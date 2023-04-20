package proxy

import (
	"net/http"
	"synapsor/pkg/plugins"
	grpcPool "synapsor/pkg/plugins/pool/grpc"
)

type Request struct {
	Header    http.Header
	Method    string
	To        string
	Query     interface{}
	TimeOut   int
	CacheTime int
}

type Plugin plugins.Plugin

// k8s vs server
func (plugin *Plugin) VsServer() {
	defer func() {
		plugin.Status <- false
	}()
	// init vs grpc pool
	grpcPool.InitGrpcConnPool()
}
