package middleware

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// 使用 Cookie 保存 session
func UseCookieSession() gin.HandlerFunc {
	store := cookie.NewStore([]byte("secret"))
	return sessions.Sessions("TICKET", store)
}

// auth 中间件
func AuthMiddle() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionValue := session.Get("userId")
		if sessionValue == nil {
			c.HTML(http.StatusOK, "login.tmpl", gin.H{
				"title": "用户登录",
			})
			c.Abort()
			return
		}
		// 设置简单的变量
		c.Set("userId", sessionValue.(uint64))

		c.Next()
		return
	}
}

// 注册和登陆时都需要保存seesion信息
func SaveAuthSession(c *gin.Context, id uint64) {
	session := sessions.Default(c)
	session.Set("userId", id)
	session.Save()
}

// 退出时清除session
func ClearAuthSession(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
}

func HasSession(c *gin.Context) bool {
	session := sessions.Default(c)
	if sessionValue := session.Get("userId"); sessionValue == nil {
		return false
	}
	return true
}

func GetSessionUserId(c *gin.Context) uint64 {
	session := sessions.Default(c)
	sessionValue := session.Get("userId")
	if sessionValue == nil {
		return 0
	}
	return sessionValue.(uint64)
}

// func GetUserSession(c *gin.Context) map[string]interface{} {

//     hasSession := HasSession(c)
//     userName := ""
//     if hasSession {
//         userId := GetSessionUserId(c)
//         userName = models.UserDetail(userId).Name
//     }
//     data := make(map[string]interface{})
//     data["hasSession"] = hasSession
//     data["userName"] = userName
//     return data
// }
