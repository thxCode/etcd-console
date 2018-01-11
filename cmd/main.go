package main

import (
	"flag"
	"strings"
	"time"
	"context"
	"errors"

	"github.com/kataras/iris"
	"github.com/kataras/iris/hero"
	v1WebRoutes "github.com/thxcode/etcd-console/backend/v1/web/routes"
	v1Services "github.com/thxcode/etcd-console/backend/v1/services"
	"github.com/kataras/iris/core/router"
	"github.com/thxcode/etcd-console/backend"
	"github.com/kataras/iris/middleware/pprof"
	"github.com/kataras/iris/middleware/recover"
	"github.com/coreos/etcd/embed"
	"os"
	"path/filepath"
	"github.com/iris-contrib/middleware/cors"
)

func main() {
	var (
		rootCtx, rootCancleFn = context.WithCancel(context.Background())
		advertise             string
		test                  bool
		endpoints             string
		logLevel              string
		backupDir             string
		config                string

		configuration backend.Configuration
	)
	defer rootCancleFn()

	// parse flags
	flag.StringVar(&advertise, "advertise", "0.0.0.0:8080", "The address is used for communicating etcd-console data.")
	flag.StringVar(&endpoints, "endpoints", "http://127.0.0.1:2379", "Specify using endpoints of etcd, splitting by comma.")
	flag.BoolVar(&test, "test", true, "Start with an embedding etcd or not.")
	flag.StringVar(&logLevel, "log-level", "debug", "Log level of etcd-console.")
	flag.StringVar(&backupDir, "backup-dir", filepath.Join(os.TempDir(), "etcd_console.backup"), "Where is storing the backup zip files.")
	flag.StringVar(&config, "config", "", "Specify the configuration yaml of etcd-console.")
	flag.Parse()

	// create application
	app := iris.New()
	logger := app.Logger()
	logger.SetLevel(configuration.LogLevel)

	// create configuration
	if config != "" {
		configuration = backend.YAML(config)
	} else {
		if len(endpoints) == 0 || endpoints == "" {
			logger.Fatal("cannot get etcd endpoints")
		}

		configuration = backend.DefaultConfiguration()

		configuration.Advertise = advertise
		configuration.Test = test
		configuration.LogLevel = logLevel
		configuration.BackupDir = backupDir
		endpointArr := strings.Split(endpoints, ",")
		for idx, endpoint := range endpointArr {
			endpoint = strings.TrimSpace(endpoint)
			endpointArr[idx] = endpoint
		}
		configuration.Endpoints = endpointArr
	}

	// test or not
	if configuration.Test {
		embedEtcdSigsChan := make(chan interface{}, 1)

		go func(rootCtx context.Context) {
			embedCfg := embed.NewConfig()
			embedCfg.Dir = filepath.Join(os.TempDir(), "etcd_console.etcd")
			embedCfg.ForceNewCluster = true
			embedCfg.LogPkgLevels = "etcdserver=WARNING,security=WARNING,raft=WARNING"

			embedEtcd, err := embed.StartEtcd(embedCfg)
			if err != nil {
				embedEtcdSigsChan <- err
			}

			select {
			case <-embedEtcd.Server.ReadyNotify():
				embedEtcdSigsChan <- new(interface{})
			case <-time.After(60 * time.Second):
				embedEtcd.Server.Stop() // trigger a shutdown
				embedEtcdSigsChan <- errors.New("embedding etcd took too long to start")
			}
		}(rootCtx)

		sig := <-embedEtcdSigsChan
		switch sig.(type) {
		case error:
			logger.Fatal(sig)
		default:
			configuration.Endpoints = []string{"http://localhost:2379"}
			logger.Info("embedding etcd is started")
		}
	}

	// create backup dir
	if stat, err := os.Stat(configuration.BackupDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(configuration.BackupDir, os.ModePerm); err != nil {
				logger.Fatalf("backup path cannot be created, %v", err)
			}
		} else {
			logger.Fatal(err)
		}
	} else if !stat.IsDir() {
		logger.Fatal("backup path is not a directory")
	}

	// create etcd client
	etcdClient := backend.NewEtcdClient(app, configuration)

	// register services
	hero.Register(
		v1Services.NewClusterService(),
		v1Services.NewClientService(),
	)

	// config routes
	app.PartyFunc("/api/v1", func(apiV1 router.Party) {

		apiV1.Any("/cluster/{op: string}", hero.Handler(v1WebRoutes.Cluster))
		apiV1.Any("/client/{op: string}", hero.Handler(v1WebRoutes.Client))

	})

	app.Get("/health", hero.Handler(func (irisCtx iris.Context) hero.Result {
		return hero.Response{
			Object: iris.Map{"health": true},
		}
	}))

	// config middlewares
	app.Use(recover.New(), cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // allows everything, use that to change the hosts.
		AllowCredentials: true,
	}))
	app.Any("/debug/pprof/{action:path}", pprof.New())
	app.UseGlobal( func(irisCtx iris.Context) {
		irisCtx.Values().Set("etcd-console.client", etcdClient)
		irisCtx.Values().Set("etcd-console.config", configuration)
		irisCtx.Values().Set("etcd-console.ctx", rootCtx)

		irisCtx.Next()
	})

	// run app
	app.Run(
		iris.Addr(configuration.Advertise),
		iris.WithoutVersionChecker,
		iris.WithoutServerError(iris.ErrServerClosed),
		iris.WithOptimizations,
		iris.WithConfiguration(configuration.ToIrisConfiguration()),
	)
}
