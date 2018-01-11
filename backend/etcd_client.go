package backend

import (
	"time"

	v3 "github.com/coreos/etcd/clientv3"
	v2 "github.com/coreos/etcd/client"
	sv2 "github.com/Masterminds/semver"
	"github.com/kataras/iris"
	"github.com/kataras/golog"
	"github.com/gorilla/http"

	"bytes"
	"net/url"
	"fmt"
	"errors"
	"encoding/json"
)

type EtcdClient struct {
	version *sv2.Version
	client  interface{}
}

type EtcdVersion struct {
	Etcdserver  string `json:"etcdserver"`
	Etcdcluster string `json:"etcdcluster"`
}

var (
	logger *golog.Logger
)

func NewEtcdClient(app *iris.Application, config Configuration) *EtcdClient {
	logger = app.Logger()

	var (
		version     *sv2.Version
		client      interface{}
		err         error
		etcdVersion bytes.Buffer
	)

	if url, err := url.Parse(config.Endpoints[0]); err != nil {
		logger.Fatal(err)
	} else {
		url.Path = "/version"
		if _, err := http.Get(&etcdVersion, url.String(), ); err != nil {
			logger.Fatal(err)
		} else {
			logger.Info(etcdVersion.String())
		}
	}

	for true {
		var etcdVersionObj EtcdVersion
		if err := json.Unmarshal(etcdVersion.Bytes(), &etcdVersionObj); err != nil {
			logger.Fatal(err)
			break
		} else {
			version, err = sv2.NewVersion(etcdVersionObj.Etcdserver)
			if err == nil {
				break
			}
		}
	}

	for true {
		if version.Major() == 2 {
			v2Client, v2Err := v2.New(v2.Config{
				Endpoints:               config.Endpoints,
				Transport:               v2.DefaultTransport,
				HeaderTimeoutPerRequest: 5 * time.Second,
			})
			if v2Err != nil {
				err = v2Err
			}

			client = &v2Client
		} else {
			client, err = v3.New(v3.Config{
				Endpoints:   config.Endpoints,
				DialTimeout: 5 * time.Second,
			})
		}
		if err == nil {
			logger.Info("etcd client is ready")
			break
		}
		logger.Warnf("etcd client is waiting for endpoints(%v)...", config.Endpoints)
		time.Sleep(2 * time.Second)
	}

	return &EtcdClient{
		version,
		client,
	}
}

func (c *EtcdClient) V2() (*v2.Client, error) {
	if c.version.Major() == 2 {
		return c.client.(*v2.Client), nil
	}

	return nil, errors.New(fmt.Sprintf("the version of etcd is %v", c.version))
}

func (c *EtcdClient) V3() (*v3.Client, error) {
	if c.version.Major() == 3 {
		return c.client.(*v3.Client), nil
	}

	return nil, errors.New(fmt.Sprintf("the version of etcd is %v", c.version))
}

func (c *EtcdClient) Version() *sv2.Version {
	return c.version
}
