package util

import (
	"context"
	"log"
	"net/http"
)

/**
* 多协程任务管理
 */

//生成 trace jobs
func GenerateTracingJobs(pch chan<- context.Context, parent context.Context, r *http.Request, svc, traceType string, tags map[string]string) {
	//设置 context
	ctx, cancel := context.WithCancel(parent)
	//设置通道
	ch := make(chan context.Context, 0)
	go doTask(ch, ctx, r, svc, traceType, tags)
	//接受信号
	pctx := <-ch
	pch <- pctx
	//销毁资源
	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		default:
			break
		}
	}
}

//执行 trace reporter
func doTask(ch chan context.Context,parent context.Context, r *http.Request, svc, traceType string, tags map[string]string) {
	//定义 tracer, closer
	var ctx context.Context
	//选择 reporter 类别
	switch(traceType) {
	case "A":
		log.Println("job A")
		break
	case "B":
		log.Println("job B")
		go func() {}()
		break
	case "C":
		log.Println("job C")
		break
	default:
		break
	}

	log.Println("tracing job finish ...")
	ch <- ctx
}


