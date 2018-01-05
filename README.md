# etcd Console

A console supports etcd v2 and v3 by Alpine.

[![](https://img.shields.io/badge/Github-thxcode/etcd--console-orange.svg)](https://github.com/thxcode/etcd-console)&nbsp;[![](https://img.shields.io/badge/Docker_Hub-maiwj/etcd--console-orange.svg)](https://hub.docker.com/r/maiwj/etcd-console)&nbsp;[![](https://img.shields.io/docker/build/maiwj/etcd-console.svg)](https://hub.docker.com/r/maiwj/etcd-console)&nbsp;[![](https://img.shields.io/docker/pulls/maiwj/etcd-console.svg)](https://store.docker.com/community/images/maiwj/etcd-console)&nbsp;[![](https://img.shields.io/github/license/thxcode/etcd-console.svg)](https://github.com/thxcode/etcd-console)

[![](https://images.microbadger.com/badges/image/maiwj/etcd-console.svg)](https://microbadger.com/images/maiwj/etcd-console)&nbsp;[![](https://images.microbadger.com/badges/version/maiwj/etcd-console.svg)](http://microbadger.com/images/maiwj/etcd-console)&nbsp;[![](https://images.microbadger.com/badges/commit/maiwj/etcd-console.svg)](http://microbadger.com/images/maiwj/etcd-console.svg)

## References

### etcd version

- [v3.2.12](Gopkg.toml)

## How to use this image

### Running parameters

```bash
$ etcd-console -h
Usage of etcd-console:
  -api-version int
    	Specify the api version of etcd. (default 3)
  -endpoints string
    	Specify using endpoints of endpoints, if on testing, it will be setting by http://localhost:2379.
  -listen-url string
    	Specify listening URL. (default "0.0.0.0:8080")
  -test
    	Specify using embedding etcd. (default true)

```

Using `ETCD_ENDPOINTS` environment variable can also set the endpoints of etcd.

### Start an instance

To start a container, use the following:

``` bash
$ docker run -d --name test-ec maiwj/etcd-console

```

### Get list from Kubernetes Pod

``` bash
$ kubectl run --image maiwj/etcd-console:latest test

```

## License

- etcd is released under the [Apache License 2.0](https://github.com/coreos/etcd/blob/master/LICENSE)
- This image is released under the [MIT License](LICENSE)
