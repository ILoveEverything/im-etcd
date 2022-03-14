package main

import (
	"fmt"
	"github.com/wu-xie-888/im-etcd/etcd"
	"testing"
	"time"
)

var (
	name = "user"
)

func TestETCD(t *testing.T) {
	client, err := etcd.NewEtcdClient(etcd.ETCD{
		Address: []string{"127.0.0.1:2379"},
		Timeout: time.Second * 3,
	})
	if err != nil {
		t.Errorf("初始化etcd客户端失败:%v", err)
		return
	}
	defer client.Close()
	t.Run("put", func(t *testing.T) {
		address, err := client.Register(etcd.Option{
			Name:    name,
			Address: "127.0.0.1:8000",
		})
		if err != nil {
			t.Errorf("注册服务失败:%v", err)
			return
		}
		fmt.Println("注册地址:", address)
	})
	t.Run("get", func(t *testing.T) {
		discover, err := client.Discover()
		if err != nil {
			t.Errorf("获取etcd节点信息失败:%v", err)
			return
		}
		fmt.Println(discover)
	})
}
