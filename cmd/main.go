package main

import (
	"os"
	"os/signal"

	"github.com/op/go-logging"
	"flag"
	"github.com/thxcode/etcd-console/backend"
	"strings"
)

const (
	endpointsEnv = "ETCD_ENDPOINTS"
)

var (
	log = logging.MustGetLogger("etcd-console")

	listenURL  string
	apiVersion int
	test       bool
	endpoints  []string

	inEndpoints string
)

func main() {
	logging.SetBackend(
		logging.NewBackendFormatter(
			logging.NewLogBackend(os.Stdout, "", 0),
			logging.MustStringFormatter(
				`%{color}%{time:2006-01-02 15:04:05.000000} %{level:.1s} | %{longpkg} :%{color:reset} %{message}`,
			)))

	flag.StringVar(&listenURL, "listen-url", "0.0.0.0:8080", "Specify listening URL.")
	flag.BoolVar(&test, "test", true, "Specify using embedding etcd.")
	flag.IntVar(&apiVersion, "api-version", 3, "Specify the api version of etcd.")
	flag.StringVar(&inEndpoints, "endpoints", "", "Specify using endpoints of endpoints, if on testing, it will be setting by http://localhost:2379.")
	flag.Parse()

	if len(inEndpoints) == 0 || inEndpoints == "" {
		inEndpoints = os.Getenv(endpointsEnv)
		if !test && inEndpoints == "" {
			log.Fatal("cannot get etcd endpoints.")
			os.Exit(1)
		}
	}

	endpoints := strings.Split(inEndpoints, ",")
	for idx, endpoint := range endpoints {
		endpoint = strings.TrimSpace(endpoint)
		endpoints[idx] = endpoint
	}

	if test {
		log.Info("starting web server with testing...")
	} else {
		log.Info("starting web server...")
	}

	server, err := backend.Start(listenURL, test, apiVersion, endpoints)
	if err != nil {
		log.Fatalf("starting web server -> %v", err)
		os.Exit(1)
	}
	defer server.Stop()

	log.Info("started web server.")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill)
	select {
	case sig := <-sigs:
		log.Infof("shutting down server with signal %s...", sig.String())
	case <-server.StopNotify():
		log.Info("shutting down server with stop signal...")
	}

}
