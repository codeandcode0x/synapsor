package util

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

const (
	CURRENT_PAGE = 1
	PAGE_SIZE    = 1
	JOB_CLOSE    = 1
	JOB_ON       = 2
)

func InitConfig() {
	viper.SetConfigName("Config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config/")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}
}

// Pagination
type Pagination struct {
	PageSize        int   `form:"pagesize" json:"pagesize"`
	CurrentPage     int   `form:"currentpage" json:"currentpage"`
	TotalRows       int64 `json:"totalrows"`
	TotalPages      int64 `json:"totalpages"`
	PreCurrentPage  int   `form:"currentpage" json:"precurrentpage"`
	NextCurrentPage int   `form:"currentpage" json:"nextcurrentpage"`
}

// message
type Message struct {
	Code    int
	Err     error
	Message string
	Data    interface{}
	Page    Pagination
}

//send info message out
func SendMessage(c *gin.Context, msg Message) {
	// err return
	if msg.Err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    msg.Code,
			"message": msg.Err.Error(),
		})
	} else {
		if msg.Page.TotalRows > 0 {
			c.JSON(http.StatusOK, gin.H{
				"code":    msg.Code,
				"message": msg.Message,
				"data":    msg.Data,
				"page":    msg.Page,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    msg.Code,
			"message": msg.Message,
			"data":    msg.Data,
		})

	}
}

//send error message out
func SendError(c *gin.Context, msg string) {
	// err return
	c.JSON(http.StatusOK, gin.H{
		"code":    -1,
		"message": msg,
	})
}
