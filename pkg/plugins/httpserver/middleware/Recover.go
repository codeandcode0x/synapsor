package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// panic recover
func Recover(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			// print steel info
			log.Printf("panic: %v\n", r)
			debug.PrintStack()
			c.JSON(http.StatusOK, gin.H{
				"code": "-1",
				"msg":  errorToString(r),
				"data": nil,
			})
			// c.Abort()
		}
	}()
	// defer recover, continue
	c.Next()
}

// recover error info to string
func errorToString(r interface{}) string {
	switch v := r.(type) {
	case error:
		return v.Error()
	default:
		return r.(string)
	}
}
