package model

// instance entity
type MetricsData struct {
	MetricsName   string `json:"metricsName"`
	MetricsData   string `json:"metricsData"`
	MetricsModule string `json:"metricsModule"`
	BaseModel
}

// user DAO
type MetricsDAO interface{}
