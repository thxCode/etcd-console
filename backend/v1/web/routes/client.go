package routes

import (
	"context"
	"github.com/kataras/iris"
	"github.com/thxcode/etcd-console/backend/v1/services"
	"github.com/kataras/iris/hero"
	"github.com/thxcode/etcd-console/backend/v1/web/viewmodels"
	"github.com/thxcode/etcd-console/backend/v1/datamodels"
	"errors"
	"time"
	"fmt"
	"github.com/thxcode/etcd-console/backend"
)

func Client(irisCtx iris.Context, service services.ClientService, op string) hero.Result {
	var (
		start         = time.Now()
		response      = hero.Response{}
		rootCtx       = irisCtx.Values().Get("etcd-console.ctx").(context.Context)
		requestMethod = irisCtx.Method()
		kvs           []datamodels.KeyValue
		err           = errors.New("method not found")
	)

	switch op {
	case "read":
		if requestMethod == iris.MethodGet {
			kvs, err = service.Get(rootCtx, irisCtx)
		}
	case "write":
		if requestMethod == iris.MethodPost {
			kvs, err = service.Set(rootCtx, irisCtx)
		}
	case "remove":
		if requestMethod == iris.MethodDelete {
			kvs, err = service.Del(rootCtx, irisCtx)
		}
	}

	if err != nil {
		irisCtx.Application().Logger().Error(err)

		response.Code = iris.StatusInternalServerError
		response.Err = err
	} else {
		response.Object = viewmodels.ClientResponse{
			KVS:    kvs,
			Result: fmt.Sprintf("took time %v", backend.RoundDownDuration(time.Since(start), time.Millisecond)),
		}
	}

	return response
}
