package viewmodels

import "github.com/thxcode/etcd-console/backend/v1/datamodels"

type ClientSetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`

	// v2
	TTL           int32  `json:"ttl"`
	SwapWithValue string `json:"swapWithValue"`
	SwapWithIndex int32  `json:"swapWithIndex"`

	// v3
	Lease       string `json:"lease"`
	PrevKV      bool   `json:"prevKV"`
	IgnoreValue bool   `json:"ignoreValue"`
	IgnoreLease bool   `json:"ignoreLease"`
}


type ClientResponse struct {
	Result  string `json:"result"`
	KVS []datamodels.KeyValue `json:"kvs"`
}
