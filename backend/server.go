package backend

import (
	"net/http"
	"sync"
	"context"
	"os"
	"fmt"
	"github.com/op/go-logging"
	v3 "github.com/coreos/etcd/clientv3"
	v2 "github.com/coreos/etcd/client"
	"github.com/thxcode/etcd-console/backend/action"

	"time"
	"github.com/coreos/etcd/embed"
	"errors"
)

var (
	log = logging.MustGetLogger("etcd-console")
)

type Server struct {
	mutex     sync.RWMutex
	web       *http.Server
	embedEtcd *embed.Etcd

	rootCancel func()
	stopChan   chan struct{}
	doneChan   chan struct{}
}

func Start(listenURL string, test bool, apiVersion int, endpoints []string) (*Server, error) {
	var (
		client    interface{}
		clientErr error

		rootCtx, rootCancelFn = context.WithCancel(context.Background())

		isV2 = false
		mux  = http.NewServeMux()

		server = &Server{
			web:        &http.Server{Addr: listenURL, Handler: mux},
			rootCancel: rootCancelFn,
			stopChan:   make(chan struct{}),
			doneChan:   make(chan struct{}),
		}
	)

	// translate environment
	if test {
		embedEtcdSigsChan := make(chan interface{}, 1)

		go func(rootCtx context.Context, server *Server) {
			embedCfg := embed.NewConfig()
			embedCfg.Dir = fmt.Sprintf("/tmp/etcd_console_%d.etcd", time.Now().Unix())
			embedCfg.ForceNewCluster = true
			embedCfg.Debug = true

			embedEtcd, err := embed.StartEtcd(embedCfg)
			if err != nil {
				embedEtcdSigsChan <- err
			}
			server.embedEtcd = embedEtcd

			select {
			case <-embedEtcd.Server.ReadyNotify():
				embedEtcdSigsChan <- new(interface{})
			case <-time.After(60 * time.Second):
				embedEtcd.Server.Stop() // trigger a shutdown
				embedEtcdSigsChan <- errors.New("embedding etcd took too long to start.")
			}
		}(rootCtx, server)

		sig := <-embedEtcdSigsChan
		switch sig.(type) {
		case error:
			return nil, sig.(error)
		default:
			endpoints = []string{"http://localhost:2379"}
			log.Info("embedding etcd is started.")
		}
	}

	if apiVersion == 2 {
		isV2 = true
	}

	// create context
	rootCtx = context.WithValue(rootCtx, "isV2", isV2)
	rootCtx = context.WithValue(rootCtx, "endpoints", endpoints)

	for true {
		if isV2 {
			client, clientErr = v2.New(v2.Config{
				Endpoints:               endpoints,
				Transport:               v2.DefaultTransport,
				HeaderTimeoutPerRequest: 5 * time.Second,
			})
		} else {
			client, clientErr = v3.New(v3.Config{
				Endpoints:   endpoints,
				DialTimeout: 10 * time.Second,
			})
		}
		if clientErr == nil {
			log.Info("etcd client is ready.")
			rootCtx = context.WithValue(rootCtx, "client", client)
			break
		}
		log.Warningf("etcd client is waiting for endpoints(%v)...", endpoints)
		time.Sleep(2 * time.Second)
	}

	// config web
	mux.Handle("/health", &action.CtxHandlerWrapper{
		rootCtx,
		action.CtxHandleFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

			w.WriteHeader(200)
			w.Write([]byte(`"health": "true"`))
			return nil
		}),
	})
	mux.Handle("/cluster/status", &action.CtxHandlerWrapper{
		rootCtx,
		action.CtxHandleFunc(action.ClusterStatusHandle),
	})
	//mux.Handle("/cluster/backup", &action.CtxHandlerWrapper{
	//	rootCtx,
	//	action.CtxHandleFunc(action.ClusterBackupHandle),
	//})
	mux.Handle("/client/get", &action.CtxHandlerWrapper{
		rootCtx,
		action.CtxHandleFunc(action.ClientGetHandle),
	})
	mux.Handle("/client/ls", &action.CtxHandlerWrapper{
		rootCtx,
		action.CtxHandleFunc(action.ClientV2LsHandle),
	})
	mux.Handle("/client/set", &action.CtxHandlerWrapper{
		rootCtx,
		action.CtxHandleFunc(action.ClientSetHandle),
	})
	mux.Handle("/client/lease", &action.CtxHandlerWrapper{
		rootCtx,
		action.CtxHandleFunc(action.ClientV3LeaseHandle),
	})
	mux.Handle("/client/remove", &action.CtxHandlerWrapper{
		rootCtx,
		action.CtxHandleFunc(action.ClientRemoveHandle),
	})

	// listening port and serving
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Warningf("web -> %v", err)
				os.Exit(0)
			}

			if !isV2 {
				cli := rootCtx.Value("client").(*v3.Client)
				cli.Close()
			}

			server.rootCancel()
			close(server.doneChan)
		}()

		if err := server.web.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	return server, nil
}

func (server *Server) StopNotify() <-chan struct{} {
	return server.stopChan
}

func (server *Server) Stop() {
	log.Warning("stopping server...")
	server.mutex.Lock()

	if server.web == nil {
		server.mutex.Unlock()
		return
	}

	close(server.stopChan)
	server.web.Close()
	<-server.doneChan
	if server.embedEtcd != nil {
		server.embedEtcd.Close()
	}
	server.mutex.Unlock()
	log.Warning("stopped server.")
}
