package etcd

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"go.etcd.io/etcd/client/v3"
	"net"
	"time"
)

// Registry 获取etcd客户端
type Registry interface {
	Register(Option) (string, error)        //注册
	Unregister() error                      //反注册
	Discover() (map[string][]string, error) //发现所有服务
	Watch()                                 //监控
}

// NewEtcdClient etcd客户端
func NewEtcdClient(opt ETCD) (Registry, error) {
	var etcd = ETCD{
		Address: []string{"127.0.0.1:2379"},
		Timeout: time.Second * 5,
		name:    "",
		client:  nil,
		node:    nil,
	}
	if len(opt.Address) > 0 {
		etcd.Address = opt.Address
	}
	if opt.Timeout > 0 {
		etcd.Timeout = opt.Timeout
	}
	tc, err := clientv3.New(clientv3.Config{
		Endpoints:   etcd.Address,
		DialTimeout: etcd.Timeout,
	})
	if err != nil {
		return nil, err
	}
	etcd.client = tc
	return &etcd, nil
}

//获取本地地址
func getLocalAddress() (string, error) {
	var addr string
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			return
		}
	}(conn)
	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if ok {
		addr = localAddr.IP.String()
		return addr, nil
	}
	return "", errors.New("获取本地地址失败！")
}

// GetFreePort 获取随机端口
func getFreePort() (port string, err error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return
	}
	defer func(l *net.TCPListener) {
		err := l.Close()
		if err != nil {
			return
		}
	}(l)
	_, port, err = net.SplitHostPort(l.Addr().String())
	return
}

func decode(val Option) []byte {
	bytes, _ := json.Marshal(val)
	return bytes
}

func encode(val []byte) Option {
	var n Option
	_ = json.Unmarshal(val, &n)
	return n
}

func joinKey(name string) string {
	return prefix + "/" + name + "/" + uuid.New().String()
}
