package services

import (
	"github.com/kataras/iris"
	"context"
	"github.com/thxcode/etcd-console/backend/v1/datamodels"
	"github.com/thxcode/etcd-console/backend"
	"time"

	"errors"
	"fmt"
	v3 "github.com/coreos/etcd/clientv3"
	"strings"
	"github.com/thxcode/etcd-console/backend/v1/web/viewmodels"
	"strconv"
)

type ClientService interface {
	Get(ctx context.Context, irisCtx iris.Context) ([]datamodels.KeyValue, error)
	Set(ctx context.Context, irisCtx iris.Context) ([]datamodels.KeyValue, error)
	Del(ctx context.Context, irisCtx iris.Context) ([]datamodels.KeyValue, error)
}

type clientService struct {
}

func NewClientService() ClientService {
	return &clientService{
	}
}

func (c *clientService) Get(ctx context.Context, irisCtx iris.Context) ([]datamodels.KeyValue, error) {
	etcdClient := irisCtx.Values().Get("etcd-console.client").(*backend.EtcdClient)

	timeout, err := irisCtx.URLParamInt64Default("timeout", 5)
	if err != nil {
		timeout = 5
	}
	timeoutCtx, timeoutCancelFn := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer timeoutCancelFn()

	var retKeyValues []datamodels.KeyValue

	version := etcdClient.Version()
	if version.Major() == 2 {

		return nil, errors.New("cannot support v2 now")
	} else {
		// prefix: bool
		// fromKey: bool
		// key: string
		// consistency: string
		// range: string
		// limit: int64
		// rev: int64
		// keysOnly: bool
		// sortOrder: string
		// sortTarget: string

		// create opts
		var opts []v3.OpOption

		prefix, err := irisCtx.URLParamBool("prefix")
		if err != nil {
			prefix = false
		}

		fromKey, err := irisCtx.URLParamBool("fromKey")
		if err != nil {
			fromKey = false
		}

		if prefix && fromKey {
			return nil, errors.New(`"prefix" and "fromKey" cannot be set at the same time, choose one`)
		}

		key := irisCtx.URLParamEscape("key")

		consistency := irisCtx.URLParamDefault("consistency", "l")
		switch consistency {
		case "s":
			opts = append(opts, v3.WithSerializable())
		case "l":
		default:
			return nil, errors.New(fmt.Sprintf(`unknown "consistency" flag %s`, consistency))
		}

		if irisCtx.URLParamExists("range") {
			opts = append(opts, v3.WithRange(irisCtx.URLParam("range")))
		}

		limit, err := irisCtx.URLParamInt64Default("limit", 0)
		if err != nil {
			limit = 0
		}
		opts = append(opts, v3.WithLimit(limit))

		rev, err := irisCtx.URLParamInt64Default("rev", 0)
		if err != nil {
			rev = 0
		}
		if rev > 0 {
			opts = append(opts, v3.WithRev(rev))
		}

		sortOrder := v3.SortNone
		switch strings.ToUpper(irisCtx.URLParamDefault("sortOrder", "")) {
		case "ASCEND":
			sortOrder = v3.SortAscend
		case "DESCEND":
			sortOrder = v3.SortDescend
		case "":
			// nothing
		default:
			return nil, errors.New(fmt.Sprintf("bad sort order %v", irisCtx.URLParam("sortOrder")))
		}
		sortTarget := v3.SortByKey
		switch strings.ToUpper(irisCtx.URLParamDefault("sortTarget", "")) {
		case "CREATE":
			sortTarget = v3.SortByCreateRevision
		case "KEY":
			sortTarget = v3.SortByKey
		case "MODIFY":
			sortTarget = v3.SortByModRevision
		case "VALUE":
			sortTarget = v3.SortByValue
		case "VERSION":
			sortTarget = v3.SortByVersion
		case "":
			// nothing
		default:
			return nil, errors.New(fmt.Sprintf("bad sort target %v", irisCtx.URLParam("sortTarget")))
		}
		opts = append(opts, v3.WithSort(sortTarget, sortOrder))

		if prefix {
			if len(key) == 0 {
				key = "\x00"
				opts = append(opts, v3.WithFromKey())
			} else {
				opts = append(opts, v3.WithPrefix())
			}
		}

		if fromKey {
			if len(key) == 0 {
				key = "\x00"
			}
			opts = append(opts, v3.WithFromKey())
		}

		keysOnly, err := irisCtx.URLParamBool("keysOnly")
		if err != nil {
			keysOnly = false
		}
		if keysOnly {
			opts = append(opts, v3.WithKeysOnly())
		}

		client, err := etcdClient.V3()
		if err != nil {
			return nil, err
		}
		getResp, err := client.Get(timeoutCtx, key, opts...)
		if err != nil {
			return nil, err
		}

		if kvsSize := len(getResp.Kvs); kvsSize != 0 {
			retKeyValues = make([]datamodels.KeyValue, kvsSize)

			for idx := range getResp.Kvs {
				kv := getResp.Kvs[idx]

				retKeyValues[idx] = datamodels.KeyValue{
					Key:            string(kv.Key),
					Value:          string(kv.Value),
					CreateRevision: kv.CreateRevision,
					ModRevision:    kv.ModRevision,
					Version:        kv.Version,
					Lease:          fmt.Sprintf("%x", kv.Lease),
				}
			}
		}

	}

	return retKeyValues, nil
}

func (c *clientService) Set(ctx context.Context, irisCtx iris.Context) ([]datamodels.KeyValue, error) {
	etcdClient := irisCtx.Values().Get("etcd-console.client").(*backend.EtcdClient)

	timeout, err := irisCtx.URLParamInt64Default("timeout", 5)
	if err != nil {
		timeout = 5
	}
	timeoutCtx, timeoutCancelFn := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer timeoutCancelFn()

	var retKeyValues []datamodels.KeyValue

	version := etcdClient.Version()
	if version.Major() == 2 {

		return nil, errors.New("cannot support v2 now")
	} else {
		clientSetRequest := &viewmodels.ClientSetRequest{}
		if err := irisCtx.ReadJSON(clientSetRequest); err != nil {
			return nil, err
		}

		if len(clientSetRequest.Lease) == 0 || clientSetRequest.Lease == "" {
			clientSetRequest.Lease = "0"
		}
		leaseId, err := strconv.ParseInt(clientSetRequest.Lease, 16, 64)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("bad lease ID (%v), expecting ID in Hex", clientSetRequest.Lease))
		}

		var opts []v3.OpOption
		if leaseId != 0 {
			opts = append(opts, v3.WithLease(v3.LeaseID(leaseId)))
		}

		if clientSetRequest.PrevKV {
			opts = append(opts, v3.WithPrevKV())
		}

		if clientSetRequest.IgnoreValue {
			opts = append(opts, v3.WithIgnoreValue())
		}

		if clientSetRequest.IgnoreLease {
			opts = append(opts, v3.WithIgnoreLease())
		}

		client, err := etcdClient.V3()
		if err != nil {
			return nil, err
		}

		setResp, err := client.Put(timeoutCtx, clientSetRequest.Key, clientSetRequest.Value, opts...)
		if err != nil {
			return nil, err
		}

		if clientSetRequest.PrevKV {
			retKeyValues = make([]datamodels.KeyValue, 1)

			respKV := setResp.PrevKv
			if clientSetRequest.IgnoreValue {
				retKeyValues[0] = datamodels.KeyValue{
					Key:            string(respKV.Key),
					CreateRevision: respKV.CreateRevision,
					ModRevision:    respKV.ModRevision,
					Version:        respKV.Version,
					Lease:          fmt.Sprintf("%x", respKV.Lease),
				}
			} else {
				retKeyValues[0] = datamodels.KeyValue{
					Key:            string(respKV.Key),
					Value:          string(respKV.Value),
					CreateRevision: respKV.CreateRevision,
					ModRevision:    respKV.ModRevision,
					Version:        respKV.Version,
					Lease:          fmt.Sprintf("%x", respKV.Lease),
				}
			}
		}

	}

	return retKeyValues, nil
}

func (c *clientService) Del(ctx context.Context, irisCtx iris.Context) ([]datamodels.KeyValue, error) {
	etcdClient := irisCtx.Values().Get("etcd-console.client").(*backend.EtcdClient)

	timeout, err := irisCtx.URLParamInt64Default("timeout", 5)
	if err != nil {
		timeout = 5
	}
	timeoutCtx, timeoutCancelFn := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer timeoutCancelFn()

	var retKeyValues []datamodels.KeyValue

	version := etcdClient.Version()
	if version.Major() == 2 {
		return nil, errors.New("cannot support v2 now")
	} else {
		// prefix: bool
		// fromKey: bool
		// prevKV: bool
		// range: string

		var opts []v3.OpOption

		prefix, err := irisCtx.URLParamBool("prefix")
		if err != nil {
			prefix = false
		}

		fromKey, err := irisCtx.URLParamBool("fromKey")
		if err != nil {
			fromKey = false
		}

		if prefix && fromKey {
			return nil, errors.New(`"prefix" and "fromKey" cannot be set at the same time, choose one`)
		}

		key := irisCtx.URLParamEscape("key")

		if irisCtx.URLParamExists("range") {
			opts = append(opts, v3.WithRange(irisCtx.URLParam("range")))
		}

		if prefix {
			if len(key) == 0 {
				key = "\x00"
				opts = append(opts, v3.WithFromKey())
			} else {
				opts = append(opts, v3.WithPrefix())
			}
		}

		if fromKey {
			if len(key) == 0 {
				key = "\x00"
			}
			opts = append(opts, v3.WithFromKey())
		}

		prevKV, err := irisCtx.URLParamBool("prevKV")
		if err != nil {
			prevKV = false
		}
		if prevKV {
			opts = append(opts, v3.WithPrevKV())
		}

		client, err := etcdClient.V3()
		if err != nil {
			return nil, err
		}
		delResp, err := client.Delete(timeoutCtx, key, opts...)
		if err != nil {
			return nil, err
		}

		if prevKV {
			if prevKVSize := len(delResp.PrevKvs); prevKVSize != 0 {
				retKeyValues = make([]datamodels.KeyValue, prevKVSize)

				for idx := range delResp.PrevKvs {
					kv := delResp.PrevKvs[idx]

					retKeyValues[idx] = datamodels.KeyValue{
						Key:            string(kv.Key),
						Value:          string(kv.Value),
						CreateRevision: kv.CreateRevision,
						ModRevision:    kv.ModRevision,
						Version:        kv.Version,
						Lease:          fmt.Sprintf("%x", kv.Lease),
					}
				}
			}
		}

	}

	return retKeyValues, nil
}
