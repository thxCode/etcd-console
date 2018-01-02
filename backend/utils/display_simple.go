package utils

import (
	"fmt"
	"context"

	v3 "github.com/coreos/etcd/clientv3"
	v2 "github.com/coreos/etcd/client"
)

type DisplaySimple struct {
}

func (d *DisplaySimple) V3SprintSetResponse(c context.Context, prevKv bool, valueOnly bool, response *v3.PutResponse) []string {
	if prevKv {
		k, v := string(response.PrevKv.Key), string(response.PrevKv.Value)

		if !valueOnly {
			return []string{fmt.Sprintf("previous key: %s.", k)}
		} else {
			return []string{fmt.Sprintf("previous key: %s, previous value: %s.", k, v)}
		}
	}

	return nil
}

func (d *DisplaySimple) V3SprintDelResponse(c context.Context, prevKv bool, valueOnly bool, response *v3.DeleteResponse) []string {
	var ret []string

	if prevKv {
		for _, kv := range response.PrevKvs {
			k, v := string(kv.Key), string(kv.Value)

			if !valueOnly {
				ret = append(ret, fmt.Sprintf("previous key: %s.", k))
			} else {
				ret = append(ret, fmt.Sprintf("previous key: %s, previous value: %s.", k, v))
			}
		}
	}

	return ret
}

func (d *DisplaySimple) V3SprintLeaseTimeToLiveResponse(c context.Context, response *v3.LeaseTimeToLiveResponse, showKeys bool) []string {
	var ret []string

	if response.GrantedTTL == 0 && response.TTL == -1 {
		ret = append(ret, fmt.Sprintf("lease %016x already expired.", response.ID))
	} else {
		if showKeys {
			ks := make([]string, len(response.Keys))
			for i := range response.Keys {
				ks[i] = string(response.Keys[i])
			}
			ret = append(ret, fmt.Sprintf("lease %016x granted with TTL(%ds), remaining(%ds), attached keys(%v)", response.ID, response.GrantedTTL, response.TTL, ks))
		} else {
			ret = append(ret, fmt.Sprintf("lease %016x granted with TTL(%ds), remaining(%ds).", response.ID, response.GrantedTTL, response.TTL))
		}
	}

	return ret
}

//func (d *DisplaySimple) V3SprintLeasesResponse(c context.Context, response *v3.LeaseLeasesResponse) []string {
//	var ret []string
//
//	ret = append(ret, fmt.Sprintf("found %d leases.", len(response.Leases)))
//	for _, item := range response.Leases {
//		ret = append(ret, fmt.Sprintf("%016x", item.ID))
//	}
//
//	return ret
//}

func (d *DisplaySimple) V2SprintSetResponse(c context.Context, response *v2.Response) []string {
	return []string{fmt.Sprintf("value: %s", response.Node.Value)}
}

func (d *DisplaySimple) V2SprintDelResponse(c context.Context, response *v2.Response) []string {
	return []string{fmt.Sprintf("previous value: %s", response.PrevNode.Value)}
}

func recursiveV2SprintLsResponseFunc(c context.Context, node *v2.Node, fillPath bool) []string {
	var ret []string

	if node.Dir && fillPath {
		ret = append(ret, fmt.Sprintf("%v/", node.Key))
	} else {
		ret = append(ret, node.Key)
	}

	for _, node := range node.Nodes {
		ret = append(ret, recursiveV2SprintLsResponseFunc(c, node, fillPath)...)
	}

	return ret
}

func (d *DisplaySimple) V2SprintLsResponse(c context.Context, response *v2.Response, fillPath bool) []string {
	var ret []string

	if !response.Node.Dir {
		ret = append(ret, response.Node.Key)
	}

	for _, node := range response.Node.Nodes {
		ret = append(ret, recursiveV2SprintLsResponseFunc(c, node, fillPath)...)
	}

	return ret
}
