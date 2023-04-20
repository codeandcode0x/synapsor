package db

import (
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"synapsor/pkg/plugins/httpserver/util"
	"time"

	"github.com/go-redis/redis"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBConn
var Conn *gorm.DB
var Cache redis.Cmdable

// init
func init() {
	// init viper config
	util.InitConfig()
	if runMode := os.Getenv("RUN_MODE"); runMode == "testing" {
		// TO-DO
	} else {
		dbInit()
		cacheInit()
	}
}

// db config init
func dbConfigInit() map[string]interface{} {
	dbMap := make(map[string]interface{})
	// get db config
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = viper.GetString("DB_HOST")
	}

	// get db port
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = viper.GetString("DB_PORT")
	}

	// get db name
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = viper.GetString("DB_NAME")
	}

	// get db user
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = viper.GetString("DB_USER")
	}

	// get db passwd
	dbPasswd := os.Getenv("DB_PASSWD")
	if dbPasswd == "" {
		dbPasswd = viper.GetString("DB_PASSWD")
	}

	// get db driver
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = viper.GetString("DB_DRIVER")
	}

	// get db pool config
	var dbMaxIdleConnsNum, dbMaxOpenConnsNum, dbConnMaxLifeTimeNum int
	dbMaxIdleConns := os.Getenv("DB_MAX_IDLE_CONNS")
	if dbMaxIdleConns == "" {
		dbMaxIdleConnsNum = viper.GetInt("DB_MAX_IDLE_CONNS")
	} else {
		dbMaxIdleConnsNum, _ = strconv.Atoi(dbMaxIdleConns)
	}

	dbMaxOpenConns := os.Getenv("DB_MAX_OPEN_CONNS")
	if dbMaxOpenConns == "" {
		dbMaxOpenConnsNum = viper.GetInt("DB_MAX_OPEN_CONNS")
	} else {
		dbMaxOpenConnsNum, _ = strconv.Atoi(dbMaxOpenConns)
	}

	dbConnMaxLifeTime := os.Getenv("DB_CONN_MAX_LIFETIME")
	if dbConnMaxLifeTime == "" {
		dbConnMaxLifeTimeNum = viper.GetInt("DB_CONN_MAX_LIFETIME")
	} else {
		dbConnMaxLifeTimeNum, _ = strconv.Atoi(dbConnMaxLifeTime)
	}

	// db dsn
	dsn := dbUser + ":" + dbPasswd + "@tcp(" + dbHost + ":" + dbPort + ")"
	dbMap["dbName"] = dbName
	dbMap["dbDriver"] = dbDriver
	dbMap["dbMaxIdleConns"] = dbMaxIdleConnsNum
	dbMap["dbMaxOpenConns"] = dbMaxOpenConnsNum
	dbMap["dbConnMaxLifeTime"] = dbConnMaxLifeTimeNum
	dbMap["dsn"] = dsn
	return dbMap
}

// db init
func dbInit() {
	//db config init
	dbMap := dbConfigInit()
	dsn, dbName, dbDriver := dbMap["dsn"].(string), dbMap["dbName"].(string), dbMap["dbDriver"].(string)
	// db dsn
	var err error
	// set db log level
	logLevel := logger.Info
	dbLogMode := os.Getenv("DB_LOGMODE")
	if dbLogMode == "" {
		dbLogMode = viper.GetString("DB_LOGMODE")
		switch strings.ToLower(dbLogMode) {
		case "silent":
			logLevel = logger.Silent
		case "warn":
			logLevel = logger.Warn
		case "error":
			logLevel = logger.Error
		case "info":
			logLevel = logger.Info
		}
	}
	//db logger

	var slowThresholdNum time.Duration
	slowThreshold := os.Getenv("DB_SLOWTHRESHOLD")
	if slowThreshold == "" {
		slowThresholdNum = (time.Duration)(viper.GetInt64("DB_SLOWTHRESHOLD"))
	} else {
		slowThresholdInt64, _ := strconv.ParseInt(slowThreshold, 10, 64)
		slowThresholdNum = (time.Duration)(slowThresholdInt64)
	}

	DBLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             slowThresholdNum * time.Second, // Slow SQL threshold
			LogLevel:                  logLevel,                       // Log level
			IgnoreRecordNotFoundError: true,                           // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,                          // Disable color
		},
	)

	//select db driver
	switch strings.ToLower(dbDriver) {
	// 支持 mysql
	case "mysql":
		//get conn
		//check db
		status := checkDB(dbName, dsn+"/?charset=utf8mb4&parseTime=True&loc=Local")
		if !status {
			return
		}

		Conn, err = gorm.Open(mysql.Open(dsn+"/"+dbName+"?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{
			Logger: DBLogger,
		})

		// 支持 oracle
		// case "oracle":
		// 	log.Println("oracle")
		// 	Conn, err = gorm.Open(oracle.Open("system/oracle@127.0.0.1:56668/XE"), &gorm.Config{
		// 		Logger: DBLogger,
		// 	})
	}

	//connect err
	if err != nil {
		panic(err.Error())
	}

	log.Println("db: mysql connect successed !")

	//sql db
	sqlDB, errSql := Conn.DB()
	//err sql
	if errSql != nil {
		panic(" seting database error !")
	}
	// db setting
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)
	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	sqlDB.SetConnMaxLifetime(time.Hour)
}

// check db
func checkDB(dbName, dsn string) bool {
	Conn, _ = gorm.Open(mysql.Open(dsn))
	checkDB := "select * from information_schema.SCHEMATA where SCHEMA_NAME = '" + dbName + "'; "
	tx := Conn.Raw(checkDB)
	rows, _ := tx.Rows()
	defer rows.Close()
	checkDBError := tx.Error
	if checkDBError != nil {
		// logging.ERROR.Error(checkDBError.Error())
		return false
	}

	if !rows.Next() {
		Conn.Exec("CREATE DATABASE IF NOT EXISTS `" + dbName + "` DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_unicode_ci;")
	}

	sqlDB, sqlErr := Conn.DB()
	if sqlErr != nil {
		// logging.ERROR.Error(sqlErr.Error())
		return false
	}
	sqlDB.Close()

	return true
}

// cache init
func cacheInit() {
	cacheEnabledBool := false
	cacheEnabled := os.Getenv("CACHE_ENABLED")
	if cacheEnabled == "" {
		cacheEnabledBool = viper.GetBool("CACHE_ENABLED")
	} else {
		cacheEnabledBool, _ = strconv.ParseBool(cacheEnabled)
	}

	if cacheEnabledBool {
		redisInit()
	}
}

// redis init
func redisInit() {
	// get cache config
	// cache connect mode
	cacheConnectMode := os.Getenv("CACHE_CONNECT_MODE")
	if cacheConnectMode == "" {
		cacheConnectMode = viper.GetString("CACHE_CONNECT_MODE")
	}
	// cache address
	cacheAddress := os.Getenv("CACHE_ADDRESS")
	if cacheAddress == "" {
		cacheAddress = viper.GetString("CACHE_ADDRESS")
	}
	cacheAddressList := strings.Split(cacheAddress, ",")
	// cache passwd
	cachePasswd := os.Getenv("CACHE_PASSWD")
	if cachePasswd == "" {
		cachePasswd = viper.GetString("CACHE_PASSWD")
	}
	// cache db num
	var cacheDBNum, cachePoolSizeNum, cacheMinidleConnsNum int
	cacheDB := os.Getenv("CACHE_DB")
	if cacheDB == "" {
		cacheDBNum = viper.GetInt("CACHE_DB")
	} else {
		cacheDBNum, _ = strconv.Atoi(cacheDB)
	}
	// cache pool size
	cachePoolSize := os.Getenv("CACHE_POOL_SIZE")
	if cacheDB == "" {
		cachePoolSizeNum = viper.GetInt("CACHE_POOL_SIZE")
	} else {
		cachePoolSizeNum, _ = strconv.Atoi(cachePoolSize)
	}
	// cache minidle conns
	cacheMinidleConns := os.Getenv("CACHE_MINIDLE_CONNS")
	if cacheDB == "" {
		cacheMinidleConnsNum = viper.GetInt("CACHE_MINIDLE_CONNS")
	} else {
		cacheMinidleConnsNum, _ = strconv.Atoi(cacheMinidleConns)
	}

	// redis client
	if "cluster" == cacheConnectMode {
		Cache = redis.NewClusterClient(&redis.ClusterOptions{
			//连接信息
			Addrs:    cacheAddressList,
			Password: cachePasswd,

			//连接池容量及闲置连接数量
			PoolSize:     cachePoolSizeNum,     //连接池最大socket连接数，默认为4倍CPU数， 4 * runtime.NumCPU
			MinIdleConns: cacheMinidleConnsNum, //在启动阶段创建指定数量的Idle连接，并长期维持idle状态的连接数不少于指定数量

			//超时
			DialTimeout:  5 * time.Second, //连接建立超时时间，默认5秒。
			ReadTimeout:  3 * time.Second, //读超时，默认3秒， -1表示取消读超时
			WriteTimeout: 3 * time.Second, //写超时，默认等于读超时
			PoolTimeout:  4 * time.Second, //当所有连接都处在繁忙状态时，客户端等待可用连接的最大等待时长，默认为读超时+1秒。

			//闲置连接检查包括IdleTimeout，MaxConnAge
			IdleCheckFrequency: 60 * time.Second, //闲置连接检查的周期，默认为1分钟，-1表示不做周期性检查，只在客户端获取连接时对闲置连接进行处理。
			IdleTimeout:        5 * time.Minute,  //闲置超时，默认5分钟，-1表示取消闲置超时检查
			MaxConnAge:         0 * time.Second,  //连接存活时长，从创建开始计时，超过指定时长则关闭连接，默认为0，即不关闭存活时长较长的连接

			//命令执行失败时的重试策略
			MaxRetries:      0,                      // 命令执行失败时，最多重试多少次，默认为0即不重试
			MinRetryBackoff: 8 * time.Millisecond,   //每次计算重试间隔时间的下限，默认8毫秒，-1表示取消间隔
			MaxRetryBackoff: 512 * time.Millisecond, //每次计算重试间隔时间的上限，默认512毫秒，-1表示取消间隔

			//钩子函数
			OnConnect: func(conn *redis.Conn) error {
				return nil
			},
		})

		if cacheClusterClient, ok := Cache.(*redis.ClusterClient); ok {
			log.Printf("redis pool info: Hits=%d Misses=%d Timeouts=%d TotalConns=%d IdleConns=%d StaleConns=%d\n",
				cacheClusterClient.PoolStats().Hits,
				cacheClusterClient.PoolStats().Misses,
				cacheClusterClient.PoolStats().Timeouts,
				cacheClusterClient.PoolStats().TotalConns,
				cacheClusterClient.PoolStats().IdleConns,
				cacheClusterClient.PoolStats().StaleConns)
		}
	} else if "single" == cacheConnectMode {
		Cache = redis.NewClient(&redis.Options{
			//连接信息
			Network:  "tcp",
			Addr:     cacheAddressList[0],
			Password: cachePasswd,
			DB:       cacheDBNum,

			//连接池容量及闲置连接数量
			PoolSize:     cachePoolSizeNum,     //连接池最大socket连接数，默认为4倍CPU数， 4 * runtime.NumCPU
			MinIdleConns: cacheMinidleConnsNum, //在启动阶段创建指定数量的Idle连接，并长期维持idle状态的连接数不少于指定数量

			//超时
			DialTimeout:  5 * time.Second, //连接建立超时时间，默认5秒。
			ReadTimeout:  3 * time.Second, //读超时，默认3秒， -1表示取消读超时
			WriteTimeout: 3 * time.Second, //写超时，默认等于读超时
			PoolTimeout:  4 * time.Second, //当所有连接都处在繁忙状态时，客户端等待可用连接的最大等待时长，默认为读超时+1秒。

			//闲置连接检查包括IdleTimeout，MaxConnAge
			IdleCheckFrequency: 60 * time.Second, //闲置连接检查的周期，默认为1分钟，-1表示不做周期性检查，只在客户端获取连接时对闲置连接进行处理。
			IdleTimeout:        5 * time.Minute,  //闲置超时，默认5分钟，-1表示取消闲置超时检查
			MaxConnAge:         0 * time.Second,  //连接存活时长，从创建开始计时，超过指定时长则关闭连接，默认为0，即不关闭存活时长较长的连接

			//命令执行失败时的重试策略
			MaxRetries:      0,                      // 命令执行失败时，最多重试多少次，默认为0即不重试
			MinRetryBackoff: 8 * time.Millisecond,   //每次计算重试间隔时间的下限，默认8毫秒，-1表示取消间隔
			MaxRetryBackoff: 512 * time.Millisecond, //每次计算重试间隔时间的上限，默认512毫秒，-1表示取消间隔

			//可自定义连接函数
			Dialer: func() (net.Conn, error) {
				netDialer := &net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 5 * time.Minute,
				}
				return netDialer.Dial("tcp", cacheAddressList[0])
			},

			//钩子函数
			OnConnect: func(conn *redis.Conn) error {
				return nil
			},
		})

		if cacheClient, ok := Cache.(*redis.Client); ok {
			log.Printf("redis pool info: Hits=%d Misses=%d Timeouts=%d TotalConns=%d IdleConns=%d StaleConns=%d\n",
				cacheClient.PoolStats().Hits,
				cacheClient.PoolStats().Misses,
				cacheClient.PoolStats().Timeouts,
				cacheClient.PoolStats().TotalConns,
				cacheClient.PoolStats().IdleConns,
				cacheClient.PoolStats().StaleConns)
		}
	} else {
		log.Printf("cache: invalid cache connect mode[%s]!", cacheConnectMode)
		panic("cache: invalid cache connect mode[" + cacheConnectMode + "]!")
	}

	// defer Cache.Close()

	_, err := Cache.Ping().Result()
	if err != nil {
		panic("cache: redis connect failed !" + err.Error())
	} else {
		log.Printf("cache: redis connect succeed !")
	}
}
