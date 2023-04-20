package proxy

import (
	"fmt"
	"os"
	"strconv"
	logging "synapsor/pkg/core/log"
	"synapsor/pkg/plugins/httpserver/controller"
	"synapsor/pkg/plugins/httpserver/middleware"
	"synapsor/pkg/plugins/httpserver/util"

	// "strconv"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/valyala/fasthttp"
)

const (
	TIME_DURATION = 10
)

var routeMap map[string]string
var routerViper *viper.Viper

// http request struct
type HttpRequest struct {
	Header    http.Header
	Method    string
	To        string
	Query     interface{}
	TimeOut   int
	CacheTime int
}

/*
* http server
* endless example :
sysType := runtime.GOOS
switch sysType {
case "windows":
	r.Run(":" + serverPort)
default:
	//end server
	endless.ListenAndServe(":"+serverPort, r)
}
*/
func (plugin *Plugin) HttpServer() {
	util.InitConfig()
	defer func() {
		plugin.Status <- false
	}()
	r := gin.Default()
	definitionRoute(r)
	//get server port
	serverPort := os.Getenv("HTTP_SERVER_PORT")
	if serverPort == "" {
		serverPort = viper.GetString("HTTP_SERVER_PORT")
	}
	// log server addr
	logging.Log.Info("http server runing :" + serverPort)
	r.Run(":" + serverPort)
}

// http proxy
// http request
func (httpReq *HttpRequest) Request() (string, error) {
	var body string = ""
	var err error

	method := strings.ToUpper(httpReq.Method)
	switch method {
	case "GET":
		body, err = getRequest(httpReq.Query.(string), httpReq.TimeOut)
	case "POST":
		body, err = postRequest(httpReq.To, httpReq.Query.(url.Values), httpReq.TimeOut, httpReq.Header)
	default:
		err = errors.New("http request any method")
	}

	return body, err
}

// get request uri
func getRequestUri(c *gin.Context) string {
	c.Request.ParseForm()
	u, _ := url.Parse(c.Request.RequestURI)

	return u.Path
}

// get request url
func getRequestUrl(to string, c *gin.Context) string {
	query, method := "", c.Request.Method
	switch method {
	case "GET":
		query = c.Request.URL.RawQuery
	case "POST":
		c.Request.ParseForm()
		param := c.Request.PostForm
		if len(param) > 0 {
			query = param.Encode()
		}
	default:
		break
	}

	queryStr := to
	if query != "" {
		queryStr = fmt.Sprintf("%s?%s", to, query)
	}
	return queryStr
}

// http get
func getRequest(u string, timeOut int) (string, error) {
	timeout := time.Duration(timeOut) * time.Second

	cli := fasthttp.Client{
		MaxConnsPerHost: 200, //最大链接数
		ReadTimeout:     timeout,
	}

	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()

	req.Header.SetContentType("application/json")
	req.Header.SetMethod("GET")
	req.SetRequestURI(u)

	if err := cli.DoTimeout(req, res, timeout); err != nil {
		return "", err
	}

	body := res.Body()
	bodyStr := string(body)

	return bodyStr, nil
}

// http post
func postRequest(to string, param map[string][]string, timeOut int, header http.Header) (string, error) {
	timeout := time.Duration(timeOut) * time.Second

	cli := fasthttp.Client{
		MaxConnsPerHost: 200,     //最大链接数
		ReadTimeout:     timeout, //主动断开时间
	}

	req, res := fasthttp.AcquireRequest(), fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(res)
	}()

	for k, v := range header {
		for _, value := range v {
			req.Header.Add(k, value)
		}
	}

	req.Header.SetMethod("POST")
	req.SetRequestURI(to)
	args := req.PostArgs()
	for k, v := range param {
		for _, value := range v {
			args.Add(k, value)
		}
	}

	if err := cli.DoTimeout(req, res, timeout); err != nil {
		return "", err
	}

	body := res.Body()
	bodyStr := string(body)

	return bodyStr, nil
}

// gin
// run
func runProxy(c *gin.Context, to string) {
	result := make(chan string)
	resultErr := make(chan error)
	var queryStr interface{}

	queryStr = getRequestUrl(to, c)
	if strings.ToUpper(c.Request.Method) == "POST" {
		c.Request.ParseForm()
		queryStr = c.Request.PostForm
	}

	httpReq := HttpRequest{
		Header:    c.Request.Header,
		Method:    c.Request.Method,
		To:        to,
		Query:     queryStr,
		TimeOut:   10,
		CacheTime: 10,
	}

	t := time.NewTimer(30 * time.Second)

	go func() {
		bodyStr, err := httpReq.Request()
		if err != nil {
			resultErr <- err
		}
		result <- bodyStr
	}()

	select {
	case res := <-result:
		c.String(http.StatusOK, res)
	case err := <-resultErr:
		c.String(http.StatusInternalServerError, fmt.Sprintln(err))
	case <-t.C:
		c.String(http.StatusNotFound, "request time out")
	}
}

// definite route
func definitionRoute(router *gin.Engine) {
	// set run mode
	runMode := os.Getenv("HTTP_DEBUG_MODE")
	if runMode == "" {
		runMode = viper.GetString("HTTP_DEBUG_MODE")
	}
	gin.SetMode(runMode)
	// middleware
	router.Use(gin.Recovery())
	// setting time out duration
	router.Use(middleware.UseCookieSession())
	// setting panic recover
	router.Use(middleware.Recover)

	var timeOutNum time.Duration
	timeOutDuration := os.Getenv("HTTP_TIME_DURATION")
	if timeOutDuration == "" {
		timeOutNum = (time.Duration)(viper.GetInt64("HTTP_TIME_DURATION"))
	} else {
		timeOutNumInt64, _ := strconv.ParseInt(timeOutDuration, 10, 64)
		timeOutNum = (time.Duration)(timeOutNumInt64)
	}
	router.Use(middleware.TimeoutHandler(time.Second * timeOutNum))

	// metrics data api
	var metricsController *controller.MetricsController
	// metrics api
	router.GET("/proxy/metricsdata", metricsController.GetPoolMetricsData)
	// no route
	router.NoRoute(noRouteResponse)
	// add route fist
	getRouterTask(router)
	// watch route
	watchRouter(router)
}

// watch router
func watchRouter(r *gin.Engine) {
	routerViper.WatchConfig()
	routerViper.OnConfigChange(func(e fsnotify.Event) {
		time.Sleep(time.Second * 1)
		logging.Log.Info("router config reload ...", e.Name)
		getRouterTask(r)
	})
}

// get router task
func getRouterTask(r *gin.Engine) {
	rvRoot := routerViper.AllSettings()
	rvMap := rvRoot["route"]
	if rvMap == nil {
		return
	}

	for _, v := range rvMap.([]interface{}) {
		addRoute(v, r)
	}
}

// add route
func addRoute(v interface{}, r *gin.Engine) {
	rmap := v.(map[string]interface{})
	rmapStr := make(map[string]string)
	for k, v := range rmap {
		strKey := fmt.Sprintf("%v", k)
		strValue := fmt.Sprintf("%v", v)
		rmapStr[strKey] = strValue
	}

	if _, exist := routeMap[rmapStr["path"]]; exist {
		logging.ERROR.Error("error: route ", rmapStr["path"], " exist !")
		return
	} else {
		routeMap[rmapStr["path"]] = rmapStr["to"]
	}

	// get method
	if rmap["method"] == "get" {
		r.GET(rmapStr["path"], func(c *gin.Context) {
			runProxy(c, rmapStr["to"])
		})
	}

	// post method
	if rmap["method"] == "post" {
		r.POST(rmapStr["path"], func(c *gin.Context) {
			runProxy(c, rmapStr["to"])
		})
	}
}

// no route
func noRouteResponse(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"code":  404,
		"error": "oops, page not exists!",
	})
}
