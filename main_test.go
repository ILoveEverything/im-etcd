package main

import (
	"fmt"
	"github.com/wu-xie-888/im-etcd/etcd"
	"testing"
	"time"
)

var (
	stop = make(chan bool, 1)
	name = "user"
)

func TestETCD(t *testing.T) {
	go func() {
		time.Sleep(time.Minute)
		stop <- true
	}()
	client, err := etcd.NewEtcdClient(etcd.ETCD{
		Address: []string{"127.0.0.1:2379"},
		Timeout: time.Second * 3,
	})
	if err != nil {
		t.Errorf("初始化etcd客户端失败:%v", err)
		return
	}
	address, err := client.Register(etcd.Option{
		Name:    name,
		Address: "127.0.0.1:8000",
	})
	if err != nil {
		fmt.Println("注册服务失败:", err)
		return
	}
	fmt.Println("注册地址:", address)
	time.Sleep(time.Second * 30)
	discover, err := client.Discover()
	if err != nil {
		fmt.Println("获取etcd节点信息失败:", err)
		return
	}
	fmt.Println(discover)
	<-stop
}
