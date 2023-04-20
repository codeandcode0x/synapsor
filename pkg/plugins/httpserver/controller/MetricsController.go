package controller

import (
	"net/http"
	"os"
	"synapsor/pkg/plugins/httpserver/service"
	"synapsor/pkg/plugins/httpserver/util"

	"github.com/gin-gonic/gin"
)

//controller struct
type MetricsController struct {
	apiVersion string
	Service    *service.MetricsService
}

//get controller
func (uc *MetricsController) getCtl() *MetricsController {
	var svc *service.MetricsService
	return &MetricsController{"v1", svc}
}

//get pool metrics data
func (uc *MetricsController) GetPoolMetricsData(c *gin.Context) {

	mDatas, err := uc.getCtl().Service.GetMetricsData()
	// error
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": err,
		})
		return
	}
	// no data
	if len(mDatas) < 1 {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "no data",
		})
		return
	}

	instanceId := os.Getenv("PROXY_INSTANCE_ID")
	if instanceId == "" {
		instanceId = "synapsor"
	}
	// send message
	util.SendMessage(c, util.Message{
		Code:    0,
		Message: "OK",
		Data: map[string]interface{}{
			"metrics":         mDatas,
			"proxyInstanceId": instanceId,
		},
	})
}
