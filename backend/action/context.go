package action

import (
	"context"
	"net/http"

	"github.com/op/go-logging"
)

var (
	log = logging.MustGetLogger("etcd-console")
)

type CtxHandler interface {
	ServeHTTPCtx(context.Context, http.ResponseWriter, *http.Request) error
}

type CtxHandleFunc func(context.Context, http.ResponseWriter, *http.Request) error

func (f CtxHandleFunc) ServeHTTPCtx(c context.Context, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("content-type","application/json")
	return f(c, w, r)
}

type CtxHandlerWrapper struct {
	Ctx		context.Context
	Handler	CtxHandler
}

func (cw CtxHandlerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := cw.Handler.ServeHTTPCtx(cw.Ctx, w, r); e != nil {
		log.Warningf("ServeHTTP [%q %q] -> %v", r.Method, r.URL.Path, e)
	}
}


