//main
package main

import (
	logging "synapsor/pkg/core/log"
	"synapsor/pkg/plugins"
	"synapsor/pkg/plugins/httpserver/util"
	"synapsor/pkg/plugins/metrics"
	"synapsor/pkg/plugins/proxy"
)

// main
func main() {
	//setting runtime cpu
	logging.Log.Info("gateway running with random balance and go v1.7 ....")
	//init config
	util.InitConfig()
	var httpPlugin, gRPCPlugin, vsPlugin proxy.Plugin
	var metricsPlugin metrics.Plugin
	//register gRPC 、HTTP、WS Server
	httpPluginChan := registerPlugins(httpPlugin.HttpServer)
	gRPCPluginChan := registerPlugins(gRPCPlugin.GRPCServer)
	//register Metrics Data Server
	metricsPluginChan := registerPlugins(metricsPlugin.ShowMetrics)
	vsPluginChan := registerPlugins(vsPlugin.VsServer)
	// return status chan
	<-httpPluginChan
	<-gRPCPluginChan
	<-vsPluginChan
	<-metricsPluginChan
}

// register plugins
func registerPlugins(factory func()) chan bool {
	var rpcProxy plugins.Plugin
	rpcProxy.Factory = factory
	rpcProxy.Status = make(chan bool)
	go rpcProxy.Factory()
	return rpcProxy.Status
}
