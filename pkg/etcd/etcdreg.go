package etcd

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	log "github.com/sirupsen/logrus"
	"time"
)

//注册租约服务
type ServiceReg struct {
	client        *clientv3.Client
	lease         clientv3.Lease //租约
	leaseResp     *clientv3.LeaseGrantResponse
	canclefunc    func()
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
}

func NewServiceReg(addr []string, timeNum int64) (*ServiceReg, error) {
	var (
		err    error
		client *clientv3.Client
	)
	// 实例化client对象，连接etcd集群
	if client, err = clientv3.New(clientv3.Config{
		Endpoints:   addr,
		DialTimeout: 5 * time.Second,
	}); err != nil {
		return nil, err
	}

	// 将当前客户端的一些信息保存到结构体中
	ser := &ServiceReg{
		client: client,
	}

	// 创建租约
	if err := ser.setLease(timeNum); err != nil {
		return nil, err
	}

	// 开启一个协程，检测租约状态，如果发现有过期的则输出Error日志
	go ser.ListenLeaseRespChan()
	return ser, nil
}

//设置租约
func (this *ServiceReg) setLease(timeNum int64) error {
	//实例化一个租约对象
	//返回一个接口类型，包含了创建、撤销、获取租约信息等方法
	lease := clientv3.NewLease(this.client)

	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	// 创建租约方法
	// timeNum: 心跳检测时间间隔
	// 当etcd服务器在给定 time-to-live(timeNum) 时间内没有接收到 keepAlive 时, 租约过期。如果租约过期则所有附加在租约上的 key 将过期并被删除
	leaseResp, err := lease.Grant(ctx, timeNum)
	if err != nil {
		cancel()
		return err
	}

	ctx, cancelFunc := context.WithCancel(context.TODO())
	// 续约指定ID
	leaseRespChan, err := lease.KeepAlive(ctx, leaseResp.ID)

	if err != nil {
		return err
	}

	this.lease = lease
	this.leaseResp = leaseResp
	this.canclefunc = cancelFunc
	this.keepAliveChan = leaseRespChan
	return nil
}

//监听续租情况
func (this *ServiceReg) ListenLeaseRespChan() {
	for {
		select {
		case leaseKeepResp := <-this.keepAliveChan:
			if leaseKeepResp == nil {
				log.Error("已经关闭续租功能")
				return
			} else {
				//log.Info("续租成功")
			}
		}
	}
}

//注册租约
func (this *ServiceReg) PutService(key, val string) error {
	//将当前客户端的ip:端口号作为K-V存储到etcd中
	//并绑定到指定的租约ID中 clientv3.WithLease(this.leaseResp.ID)
	//在租约过期时，etcd会主动触发删除key操作
	kv := clientv3.NewKV(this.client)
	_, err := kv.Put(context.TODO(), key, val, clientv3.WithLease(this.leaseResp.ID))
	return err
}

//撤销租约
func (this *ServiceReg) RevokeLease() error {
	this.canclefunc()
	time.Sleep(2 * time.Second)
	_, err := this.lease.Revoke(context.TODO(), this.leaseResp.ID)
	return err
}
