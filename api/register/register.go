package register

import (
	"encoding/json"
	"github.com/woodylan/go-websocket/api"
	"github.com/woodylan/go-websocket/define/retcode"
	"github.com/woodylan/go-websocket/servers"
	"net/http"
)

type Controller struct {
}

type inputData struct {
	// 用户ID
	SystemId string `json:"systemId" validate:"required"`
}

func (c *Controller) Run(w http.ResponseWriter, r *http.Request) {
	var inputData inputData
	// Decode代替unmarshel
	// 使用json.Decoder，如果你的数据从一个即将io.Reader流，或者需要多个值，从数据流进行解码。
	// 使用json.Unmarshal如果你已经在内存中的JSON数据。
	if err := json.NewDecoder(r.Body).Decode(&inputData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 验证参数
	err := api.Validate(inputData)
	if err != nil {
		api.Render(w, retcode.FAIL, err.Error(), []string{})
		return
	}

	// 注册用户ID
	// 将用户ID保存起来
	// 1.集群模式下保存到etcd中; 2.单机模式下直接用sync.map保存在内存中
	err = servers.Register(inputData.SystemId)
	if err != nil {
		api.Render(w, retcode.FAIL, err.Error(), []string{})
		return
	}

	api.Render(w, retcode.SUCCESS, "success", []string{})
	return
}
