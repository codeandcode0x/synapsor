package grpc

import (
	"context"
	"errors"
	logging "synapsor/pkg/core/log"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// 连接池初始化出错
var ErrPoolInit = errors.New("Pool init error")

// 连接池模型
const (
	STRICT_MODE = iota
	LOOSE_MODE
	STRICT_NETWORK_MODE = 1
	GLOBAL_NETWORK_MODE = 2
)

// Pool 连接池
type Pool struct {
	name                    string // 连接池名称
	proxyModel              string // 连接池负载模式
	code                    string // 连接池 code
	clients                 chan *Client
	connCurrent             int32         // 当前连接数
	capacity                int32         // 容量
	weight                  int32         // 权重
	size                    int32         // 容量大小 (动态变化)
	idleDur                 time.Duration // 空闲时间
	maxLifeDur              time.Duration // 最大连接时间
	timeout                 time.Duration // Pool 的关闭超时时间
	factor                  Factory       // gRPC 工厂函数
	lock                    sync.RWMutex  // 读写锁
	mode                    int           // 连接池 模型
	poolRemoteAddr          string        // 远程连接地址
	averageRequestTimeTotal int64         // 耗时统计
	averageRequestTime      int64         // 平均耗时
	averageRequestTimeNum   int64         // 耗时计算数量单元
	sumRequestTimes         int64         // 总请求次数
	status                  bool          // 是否可用
}

// Client 封装的 grpc.ClientConn
type Client struct {
	*grpc.ClientConn
	timeUsed time.Time
	timeInit time.Time
	pool     *Pool
}

// gRPC 连接工厂方法
type Factory func() (*grpc.ClientConn, error)

// stream director
type StreamDirector func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, *Client, error)

// 创建连接池
func NewPool(factor Factory, init, size int32, idleDur, maxLifeDur, timeout time.Duration, mode int) (*Pool, error) {
	return initPool(factor, init, size, idleDur, maxLifeDur, timeout, mode)
}

// 初始化连接池
func initPool(factor Factory, init, size int32, idleDur, maxLifeDur, timeout time.Duration, mode int) (*Pool, error) {
	// 参数验证
	if factor == nil {
		return nil, ErrPoolInit
	}
	if init < 0 || size <= 0 || idleDur < 0 || maxLifeDur < 0 {
		return nil, ErrPoolInit
	}
	// init pool
	if init > size {
		init = size
	}
	pool := &Pool{
		clients:    make(chan *Client, size),
		size:       size,
		capacity:   size,
		idleDur:    idleDur,
		maxLifeDur: maxLifeDur,
		timeout:    timeout,
		factor:     factor,
		mode:       mode,
		status:     true,
	}
	// init client
	for i := int32(0); i < init; i++ {
		client, err := pool.createClient()
		if err != nil {
			return nil, ErrPoolInit
		}
		pool.clients <- client
	}
	return pool, nil
}

// 创建连接
func (pool *Pool) createClient() (*Client, error) {
	conn, err := pool.factor()
	if err != nil {
		return nil, ErrPoolInit
	}
	now := time.Now()
	client := &Client{
		ClientConn: conn,
		timeUsed:   now,
		timeInit:   now,
		pool:       pool,
	}
	// atomic.AddInt32(&pool.connCurrent, 1)
	return client, nil
}

// 从连接池取出一个连接
func (pool *Pool) Acquire(ctx context.Context) (*Client, error) {
	if pool.IsClose() {
		return nil, errors.New("Pool is closed")
	}

	// defer func() {
	// 	atomic.AddInt32(&pool.connCurrent, 1)
	// }()

	var client *Client
	now := time.Now()
	select {
	case <-ctx.Done():
		if pool.mode == STRICT_MODE {
			logging.Log.Info("ctx done after client close !")
			return client, nil
		} else if pool.mode == LOOSE_MODE {
			var err error
			if pool.GetConnCurrent() > int32(pool.capacity) && pool.GetConnCurrent() <= 5*int32(pool.capacity) {
				client, err = pool.createClient()
				pool.clients <- client
			}
			return <-pool.clients, err
		}
	case client = <-pool.clients:
		// per request time
		if client != nil && pool.idleDur > 0 && client.timeUsed.Add(pool.idleDur).After(now) {
			client.timeUsed = now
			return client, nil
		}
	}
	// 如果连接已经是idle连接，或者是非严格模式下没有获取连接
	// 则新建一个连接同时销毁原有idle连接
	if client != nil {
		client.Destory()
	}
	client, err := pool.createClient()
	if err != nil {
		return nil, err
	}
	return client, nil
}

// 连接池关闭
func (pool *Pool) Close() {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	if pool.IsClose() {
		return
	}

	clients := pool.clients
	pool.clients = nil

	// 异步处理池里的连接
	go func() {
		for len(clients) > 0 {
			client := <-clients
			if client != nil {
				client.Destory()
			}
		}
	}()

	pool.status = false
}

// 连接池是否关闭
func (pool *Pool) IsClose() bool {
	return pool == nil || pool.clients == nil
}

// 连接池中连接数
func (pool *Pool) Size() int {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	return len(pool.clients)
}

// 实际连接数
func (pool *Pool) GetConnCurrent() int32 {
	return pool.capacity - int32(pool.Size())
}

// 连接关闭
func (client *Client) Close() {
	go func() {
		pool := client.pool
		now := time.Now()
		// 连接池关闭了直接销毁
		if pool.IsClose() {
			client.Destory()
			return
		}
		// 如果连接存活时间超长也直接销毁连接
		// if pool.maxLifeDur > 0 && client.timeInit.Add(pool.maxLifeDur).Before(now) {
		// 	client.Destory()
		// 	return
		// }

		if client.ClientConn == nil {
			return
		}
		client.timeUsed = now
		client.pool.clients <- client

	}()
}

// 销毁 client
func (client *Client) Destory() {
	if client.ClientConn != nil {
		client.ClientConn.Close()
		// atomic.AddInt32(&client.pool.connCurrent, -1)
	}
	client.ClientConn = nil
	client.pool = nil
}

// 获取连接创建时间
func (client *Client) TimeInit() time.Time {
	return client.timeInit
}

// 获取连接上一次使用时间
func (client *Client) TimeUsed() time.Time {
	return client.timeUsed
}
