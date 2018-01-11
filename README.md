# etcd Console

A console supports etcd v3 by Alpine.

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
  -advertise string
        The address is used for communicating etcd-console data. (default "0.0.0.0:8080")
  -backup-dir string
        Where is storing the backup zip files. (default "/${os.TempDir()}/etcd_console.backup")
  -config string
        Specify the configuration yaml of etcd-console.
  -endpoints string
        Specify using endpoints of etcd, splitting by comma. (default "http://127.0.0.1:2379")
  -log-level string
        Log level of etcd-console. (default "debug")
  -test
        Start with an embedding etcd or not. (default true)

```

### Start an instance

To start a container, use the following:

``` bash
$ docker run -d --name test-ec -p 8080:80 maiwj/etcd-console

```

### Get list from Kubernetes Pod

``` bash
$ kubectl run --image maiwj/etcd-console:latest --port=80 test

```

## License

- etcd is released under the [Apache License 2.0](https://github.com/coreos/etcd/blob/master/LICENSE)
- This image is released under the [MIT License](LICENSE)
