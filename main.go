package main

import (
	"fmt"
	"github.com/woodylan/go-websocket/define"
	"github.com/woodylan/go-websocket/pkg/etcd"
	"github.com/woodylan/go-websocket/pkg/setting"
	"github.com/woodylan/go-websocket/routers"
	"github.com/woodylan/go-websocket/servers"
	"github.com/woodylan/go-websocket/tools/log"
	"github.com/woodylan/go-websocket/tools/util"
	"net"
	"net/http"
)

func init() {
	setting.Setup()
	log.Setup()
}

func main() {
	//初始化RPC服务
	initRPCServer()

	//将服务器地址、端口注册到etcd中
	registerServer()

	//初始化路由
	routers.Init()

	//启动一个定时器用来发送心跳
	servers.PingTimer()

	fmt.Printf("服务器启动成功，端口号：%s\n", setting.CommonSetting.HttpPort)

	if err := http.ListenAndServe(":"+setting.CommonSetting.HttpPort, nil); err != nil {
		panic(err)
	}
}

func initRPCServer() {
	//如果是集群，则启用RPC进行通讯
	if util.IsCluster() {
		//初始化RPC服务
		servers.InitGRpcServer()
		fmt.Printf("启动RPC，端口号：%s\n", setting.CommonSetting.RPCPort)
	}
}

//ETCD注册发现服务
func registerServer() {
	if util.IsCluster() {
		//注册租约
		//租约的作用是检测此客户端存活状态的机制, 如果etcd检测出此客户端在指定TTL时间内未收到keepAlive
		//则表示租约到期，客户端挂掉
		//将租约绑定到K-V中，每个客户端代表一个K-V，如果租约到期后，触发K-V删除操作，将此客户端踢出集群中
		ser, err := etcd.NewServiceReg(setting.EtcdSetting.Endpoints, 5)
		if err != nil {
			panic(err)
		}

		// 获取本机IP和端口信息，作为etcd的键(key)
		hostPort := net.JoinHostPort(setting.GlobalSetting.LocalHost, setting.CommonSetting.RPCPort)
		//添加key
		//将本机客户端地址添加到etcd中
		err = ser.PutService(define.ETCD_SERVER_LIST+hostPort, hostPort)
		if err != nil {
			panic(err)
		}

		// 连接etcd
		cli, err := etcd.NewClientDis(setting.EtcdSetting.Endpoints)
		if err != nil {
			panic(err)
		}
		// 获取集群内所有服务器地址
		_, err = cli.GetService(define.ETCD_SERVER_LIST)
		if err != nil {
			panic(err)
		}
	}
}
