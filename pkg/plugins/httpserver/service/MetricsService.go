package service

import (
	"synapsor/pkg/plugins/httpserver/model"
	"synapsor/pkg/plugins/metrics"
)

type MetricsService struct {
	DAO model.MetricsDAO
}

// get user service
func (s *MetricsService) getSvc() *MetricsService {
	var m model.BaseModel
	return &MetricsService{
		DAO: &model.MetricsData{BaseModel: m},
	}
}

func (s *MetricsService) GetMetricsData() (map[string]map[string]int, error) {
	return metrics.PoolMetrics(), nil
}
