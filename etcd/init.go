package etcd

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/client/v3"
	"net"
)

func (e *ETCD) Register(opt Option) (string, error) {
	if e.isNull() {
		return "", errClientNotExist
	}
	if len(opt.Address) > 0 {
		host, port, err := defaultAddress(opt.Address)
		if err != nil {
			return "", err
		}
		address := net.JoinHostPort(host, port)
		opt.Address = address
	} else {
		host, err := getLocalAddress()
		if err != nil {
			return "", err
		}
		port, err := getFreePort()
		if err != nil {
			return "", err
		}
		opt.Address = net.JoinHostPort(host, port)
	}
	err := e.registerNode(joinKey(opt.Name), opt)
	if err != nil {
		return "", err
	}
	e.name = joinKey(opt.Name)
	if err != nil {
		return "", err
	}
	go e.Watch()
	return opt.Address, nil
}

func (e *ETCD) Unregister() error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	_, err := e.client.Delete(ctx, e.name)
	if err != nil {
		return err
	}
	return nil
}

func (e *ETCD) Discover() (map[string][]string, error) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	get, err := e.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	for _, kv := range get.Kvs {
		if kv != nil {
			e.lock.Lock()
			if e.node != nil {
				e.node[string(kv.Key)] = encode(kv.Value)
			} else {
				e.node = make(map[string]Option)
				e.node[string(kv.Key)] = encode(kv.Value)
			}
			e.lock.Unlock()
		}
	}
	var list = make(map[string][]string)
	e.lock.Lock()
	for _, opt := range e.node {
		addr, ok := list[opt.Name]
		if ok {
			addr = append(addr, opt.Address)
			list[opt.Name] = addr
		} else {
			as := make([]string, 0)
			as = append(as, opt.Address)
			list[opt.Name] = as
		}
	}
	e.lock.Unlock()
	return list, nil
}

func (e *ETCD) Watch() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	if e.isNull() {
		fmt.Println(errClientNotExist)
		return
	}
	for {
		watch := e.client.Watch(ctx, prefix, clientv3.WithPrefix())
		for {
			for _, v := range (<-watch).Events {
				if v != nil && v.PrevKv != nil {
					e.lock.Lock()
					if e.node != nil {
						e.node[string(v.PrevKv.Key)] = encode(v.PrevKv.Value)
					} else {
						e.node = make(map[string]Option)
						e.node[string(v.PrevKv.Key)] = encode(v.PrevKv.Value)
					}
					e.lock.Unlock()
				}
			}
		}
	}
}

//注册单节点
func (e *ETCD) registerNode(name string, opt Option) (err error) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	lease := clientv3.NewLease(e.client)
	grant, err := lease.Grant(ctx, 5)
	if err != nil {
		return err
	}
	_, err = e.client.Put(ctx, name, string(decode(opt)), clientv3.WithLease(grant.ID))
	if err != nil {
		return err
	}
	go leaseRenewal(lease, grant.ID)
	return nil
}

//判断etcd客户端是否存在
func (e *ETCD) isNull() bool {
	return e.client == nil
}

//排除本地默认地址和端口
func defaultAddress(addr string) (host string, port string, err error) {
	//切割地址
	host, port, err = net.SplitHostPort(addr)
	if err != nil {
		return
	}
	//判断地址是否合法
	if len(host) == 0 || host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" {
		host, err = getLocalAddress()
		if err != nil {
			return
		}
	}
	//判断端口是否正常
	if port == "0" {
		port, err = getFreePort()
		if err != nil {
			return
		}
	}
	return
}

//延续租约
func leaseRenewal(l clientv3.Lease, id clientv3.LeaseID) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for {
		select {
		case <-stopChan:
			return
		default:
			_, err := l.KeepAlive(ctx, id)
			if err != nil {
				fmt.Println(id, "续租失败:", err)
			}
		}
		return
		//time.Sleep(time.Second * 5)
	}
}