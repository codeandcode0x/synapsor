package metrics

import (
	"strings"
	"synapsor/pkg/plugins"
	"synapsor/pkg/plugins/httpserver/util"
	"synapsor/pkg/plugins/pool/grpc"

	"github.com/spf13/viper"
)

type Plugin plugins.Plugin

// asr metrics

func PoolMetrics() map[string]map[string]int {
	return grpc.GetConnPoolMetricsData()
}

// metrics server
func (plugin *Plugin) ShowMetrics() {
	// get gRPC port
	defer func() {
		plugin.Status <- false
	}()

	util.InitConfig()

	if strings.ToLower(viper.GetString("RUN_MODE")) == "dev" {
		/*
			for {
				time.Sleep(1 * time.Second)
				if grpc.GetZoneTimeStatus() {
					// grpc.ResetPoolsSumRequestTimes()
				}
			}
		*/
	}
}
