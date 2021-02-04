package servers

import (
	"encoding/json"
	"errors"
	"github.com/woodylan/go-websocket/define"
	"github.com/woodylan/go-websocket/pkg/etcd"
	"github.com/woodylan/go-websocket/tools/util"
	"sync"
	"time"
)

// 用户信息
type accountInfo struct {
	SystemId     string `json:"systemId"`
	RegisterTime int64  `json:"registerTime"`
}

var SystemMap sync.Map

// 注册用户id
func Register(systemId string) (err error) {
	//校验是否为空
	if len(systemId) == 0 {
		return errors.New("系统ID不能为空")
	}

	accountInfo := accountInfo{
		SystemId:     systemId,
		RegisterTime: time.Now().Unix(),
	}

	if util.IsCluster() {
		//集群部署下，将用户ID存到etcd中
		//所有集群共享用户ID信息，注册前先判断是否存在
		//判断是否被注册
		resp, err := etcd.Get(define.ETCD_PREFIX_ACCOUNT_INFO + systemId)
		if err != nil {
			return err
		}

		if resp.Count > 0 {
			return errors.New("该系统ID已被注册")
		}

		jsonBytes, _ := json.Marshal(accountInfo)

		//注册
		err = etcd.Put(define.ETCD_PREFIX_ACCOUNT_INFO+systemId, string(jsonBytes))
		if err != nil {
			panic(err)
			return err
		}
	} else {
		// 单机部署
		// 使用一个sync.map存储所有用户信息
		// key为用户ID
		if _, ok := SystemMap.Load(systemId); ok {
			return errors.New("该系统ID已被注册")
		}

		SystemMap.Store(systemId, accountInfo)
	}

	return nil
}
