package middleware

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vearne/golib/buffpool"
)

type GinWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w GinWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

// time out mid
func TimeoutHandler(t time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		buffer := buffpool.GetBuff()
		blw := &GinWriter{body: buffer, ResponseWriter: c.Writer}
		c.Writer = blw
		finish := make(chan struct{})

		go func() {
			c.Next()
			finish <- struct{}{}
		}()

		select {
		case <-time.After(t):
			c.Writer.WriteHeader(http.StatusGatewayTimeout)
			c.Set("errorCode", http.StatusGatewayTimeout)
			c.Abort()
			return
		case <-finish:
			blw.ResponseWriter.Write(buffer.Bytes())
			buffpool.PutBuff(buffer)
		}
	}
}

// request job
func requestJob(ctx context.Context, c *gin.Context, structChan chan struct{}) {
	ch := make(chan bool, 1)
	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			ch <- true
		}
		ch <- false
	}(ctx)
	c.Next()
	structChan <- struct{}{}
	if <-ch {
		c.Abort()
		return
	}
}
