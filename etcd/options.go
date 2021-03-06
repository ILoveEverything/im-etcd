package etcd

import (
	"errors"
	"go.etcd.io/etcd/client/v3"
	"sync"
	"time"
)

var (
	prefix            = "/im/mirco/server"
	errClientNotExist = errors.New("客户端不存在")
	stopChan          = make(chan bool, 1) //当从chan中取出值时,停止延时租约
)

// ETCD etcd客户端包装信息
type ETCD struct {
	Address     []string          //注册地址
	Timeout     time.Duration     //注册超时时长
	name        string            //当前服务节点名称
	client      *clientv3.Client  //etcd客户端
	node        map[string]Option //节点信息
	lock        sync.Mutex        //互斥锁
	lease       clientv3.Lease    //租约
	leaseId     clientv3.LeaseID  //续租id
	serverWatch map[string]bool   //存储进行watch监控的服务
}

// Option 服务信息
type Option struct {
	Name    string //服务名称
	Address string //服务GRPC地址
	Version string //服务版本
}
