package plugins

type Pluginer interface {
	HttpServer()
	GRPCServer()
}

type Plugin struct {
	Factory func()
	Status  chan bool
}
