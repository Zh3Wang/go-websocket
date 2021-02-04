package etcd

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	log "github.com/sirupsen/logrus"
	"github.com/woodylan/go-websocket/pkg/setting"
	"time"
)

type ClientDis struct {
	client *clientv3.Client
}

// 连接etcd
func NewClientDis(addr []string) (*ClientDis, error) {
	conf := clientv3.Config{
		Endpoints:   addr,
		DialTimeout: 5 * time.Second,
	}
	if client, err := clientv3.New(conf); err == nil {
		return &ClientDis{
			client: client,
		}, nil
	} else {
		return nil, err
	}
}

// 初始化操作
// 根据key前缀获取所有服务器的值
func (this *ClientDis) GetService(prefix string) ([]string, error) {
	//根据前缀获取指定key信息，即返回集群列表（所有服务器的地址）
	resp, err := this.client.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	// 将集群列表保存起来
	addrs := this.extractAddrs(resp)

	// 开启协程后台实时检测key变化
	go this.watcher(prefix)
	return addrs, nil
}

// 检测服务器集群列表key状态变化
func (this *ClientDis) watcher(prefix string) {
	// 带上clientv3.WithPrefix()参数
	// 表示检测符合前缀的所有key
	rch := this.client.Watch(context.Background(), prefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			// 如果有新增或删除服务器，修改本地集群列表
			switch ev.Type {
			case mvccpb.PUT:
				this.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE:
				this.DelServiceList(string(ev.Kv.Key))
			}
		}
	}
}

// 设置本地的集群列表
func (this *ClientDis) extractAddrs(resp *clientv3.GetResponse) []string {
	addrs := make([]string, 0)
	if resp == nil || resp.Kvs == nil {
		return addrs
	}
	for i := range resp.Kvs {
		if v := resp.Kvs[i].Value; v != nil {
			//将集群信息保存起来
			this.SetServiceList(string(resp.Kvs[i].Key), string(resp.Kvs[i].Value))
			addrs = append(addrs, string(v))
		}
	}
	return addrs
}

func (this *ClientDis) SetServiceList(key, val string) {
	setting.GlobalSetting.ServerListLock.Lock()
	defer setting.GlobalSetting.ServerListLock.Unlock()
	setting.GlobalSetting.ServerList[key] = val
	log.Info("发现服务：", key, " 地址:", val)
}

func (this *ClientDis) DelServiceList(key string) {
	setting.GlobalSetting.ServerListLock.Lock()
	defer setting.GlobalSetting.ServerListLock.Unlock()
	delete(setting.GlobalSetting.ServerList, key)
	log.Println("服务下线:", key)
}
