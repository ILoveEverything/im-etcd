package etcd

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
	"net"
	"time"
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
	err := e.registerNode(joinServerName(opt.Name), opt)
	if err != nil {
		return "", err
	}
	e.name = joinServerName(opt.Name)
	if err != nil {
		return "", err
	}
	return opt.Address, nil
}

func (e *ETCD) Unregister() error {
	if e.isNull() {
		return errClientNotExist
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	_, err := e.client.Delete(ctx, e.name)
	if err != nil {
		return err
	}
	return nil
}

func (e *ETCD) Discover(name string) error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	get, err := e.client.Get(ctx, joinServerPrefix(name), clientv3.WithPrefix())
	if err != nil {
		return err
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
	return nil
}

func (e *ETCD) ServerNode(name string) ([]string, error) {
	if e.isNull() {
		return nil, errClientNotExist
	}
	var list = make([]string, 0)
NODE:
	e.lock.Lock()
	for _, opt := range e.node {
		if opt.Name == name {
			list = append(list, opt.Address)
		}
	}
	e.lock.Unlock()
	if len(list) == 0 {
		err := e.Discover(name)
		if err != nil {
			return nil, err
		}
		goto NODE
	}
	b, ok := e.serverWatch[name]
	if !ok || (ok && !b) {
		go e.Watch(name)
	}
	return list, nil
}

func (e *ETCD) Watch(name string) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	watch := e.client.Watch(ctx, joinServerPrefix(name), clientv3.WithPrefix())
	for ev := range watch {
		for _, v := range ev.Events {
			if v != nil && v.Kv != nil {
				e.lock.Lock()
				if v.Type == mvccpb.PUT {
					if e.node != nil {
						e.node[string(v.Kv.Key)] = encode(v.Kv.Value)
					} else {
						e.node = make(map[string]Option)
						e.node[string(v.Kv.Key)] = encode(v.Kv.Value)
					}
				}
				if v.Type == mvccpb.DELETE {
					delete(e.node, string(v.Kv.Key))
				}
				e.lock.Unlock()
			}
		}
	}
}

func (e *ETCD) Close() {
	stopChan <- true
	_ = e.client.Close()
}

//???????????????
func (e *ETCD) registerNode(name string, opt Option) (err error) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	lease := clientv3.NewLease(e.client)
	e.lease = lease
	grant, err := lease.Grant(ctx, 5)
	if err != nil {
		return err
	}
	e.leaseId = grant.ID
	_, err = e.client.Put(ctx, name, string(decode(opt)), clientv3.WithLease(grant.ID))
	if err != nil {
		return err
	}
	go e.leaseRenewal()
	return nil
}

//??????etcd?????????????????????
func (e *ETCD) isNull() bool {
	return e.client == nil
}

//?????????????????????????????????
func defaultAddress(addr string) (host string, port string, err error) {
	//????????????
	host, port, err = net.SplitHostPort(addr)
	if err != nil {
		return
	}
	//????????????????????????
	if len(host) == 0 || host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" {
		host, err = getLocalAddress()
		if err != nil {
			return
		}
	}
	//????????????????????????
	if port == "0" {
		port, err = getFreePort()
		if err != nil {
			return
		}
	}
	return
}

//????????????
func (e *ETCD) leaseRenewal() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	for {
		select {
		case <-stopChan:
			return
		default:
			alive, err := e.lease.KeepAlive(ctx, e.leaseId)
			if err != nil {
				fmt.Println(e.leaseId, "????????????:", err)
			}
			for result := range alive {
				e.leaseId = result.ID
			}
		}
		time.Sleep(time.Second * e.Timeout)
	}
}
