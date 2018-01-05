package action

import (
	"net/http"
	"context"
	"encoding/json"
	"html/template"
	"time"
	"strconv"
	"fmt"

	v3 "github.com/coreos/etcd/clientv3"
	v2 "github.com/coreos/etcd/client"
	"github.com/thxcode/etcd-console/backend/utils"
	"strings"
	"qiniupkg.com/x/errors.v7"
)

const (
	minScaleDisplayUnit = time.Millisecond
)

var (
	displaySimple = &utils.DisplaySimple{}
)

func getV2Client(c context.Context) (v2.Client, error) {
	client := c.Value("client").(v2.Client)
	if client == nil {
		return nil, errors.New("v2 client create fail")
	}

	return client, nil
}

func getV3Client(c context.Context) (*v3.Client, error) {
	client := c.Value("client").(*v3.Client)
	if client == nil {
		return nil, errors.New("v3 client create fail")
	}

	return client, nil
}

type MemberStatus struct {
	Name     string
	ID       string
	Endpoint string
	IsLeader bool
	DBSize   int64
	Version  string
}

type StatusClusterResponse struct {
	MemberStatuses []MemberStatus
}

type StatusClientResponse struct {
	Success bool
	Result  string
	Results []MemberStatus
}

func ClusterStatusHandle(c context.Context, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	reqStart := time.Now()
	isV2 := c.Value("isV2").(bool)

	switch r.Method {
	case http.MethodGet:
		response := StatusClientResponse{Success: true}

		// translate parameters
		queries := r.URL.Query()

		timeoutParam := queries.Get("Timeout")
		timeout, err := time.ParseDuration(timeoutParam)
		if err != nil {
			timeout = 5
		}
		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
		defer timeoutCancelFn()

		if isV2 {
			// call etcd
			v2Client, err := getV2Client(c)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}
			v2ClientApi := v2.NewMembersAPI(v2Client)

			var (
				etcdMembersChan = make(chan []v2.Member)
				etcdLeaderChan  = make(chan v2.Member)
				errorMemberChan = make(chan error)
				errorLeaderChan = make(chan error)
			)

			go func(timeoutCtx context.Context) {
				etcdMembers, err := v2ClientApi.List(timeoutCtx)
				if err != nil {
					errorMemberChan <- err
				}
				etcdMembersChan <- etcdMembers
			}(timeoutCtx)

			go func(timeoutCtx context.Context) {
				etcdLeader, err := v2ClientApi.Leader(timeoutCtx)
				if err != nil {
					errorLeaderChan <- err
				}
				etcdLeaderChan <- *etcdLeader
			}(timeoutCtx)

			select {
			case etcdMembers := <-etcdMembersChan:
				select {
				case etcdLeader := <-etcdLeaderChan:
					var results []MemberStatus
					for _, etcdMember := range etcdMembers {
						if etcdLeader.ID == etcdMember.ID {
							results = append(results, MemberStatus{IsLeader: true, Name: etcdLeader.Name, ID: etcdLeader.ID, Endpoint: etcdLeader.ClientURLs[0]})
						} else {
							results = append(results, MemberStatus{IsLeader: false, Name: etcdMember.Name, ID: etcdMember.ID, Endpoint: etcdMember.ClientURLs[0]})
						}
					}
					response.Results = results
				case err := <-errorLeaderChan:
					response.Success = false
					response.Result = err.Error()
					return json.NewEncoder(w).Encode(response)
				}
			case err := <-errorMemberChan:
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			case <-time.After(60 * time.Second):
				response.Success = false
				response.Result = "took too long to status."
				return json.NewEncoder(w).Encode(response)
			}

		} else {
			// call etcd
			v3Client, err := getV3Client(c)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}

			etcdMemberListResponse, err := v3Client.MemberList(timeoutCtx)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}
			etcdMembers := etcdMemberListResponse.Members

			var results []MemberStatus
			for _, etcdMember := range etcdMembers {
				endpoint := etcdMember.ClientURLs[0]

				etcdStatus, err := v3Client.Status(timeoutCtx, endpoint)
				if err != nil {
					response.Success = false
					response.Result = err.Error()
					return json.NewEncoder(w).Encode(response)
				}

				if etcdStatus.Leader == etcdMember.ID {
					results = append(results, MemberStatus{IsLeader: true, Name: etcdMember.Name, ID: fmt.Sprintf("%x", etcdMember.ID), Endpoint: endpoint, DBSize: etcdStatus.DbSize, Version: etcdStatus.Version})
				} else {
					results = append(results, MemberStatus{IsLeader: false, Name: etcdMember.Name, ID: fmt.Sprintf("%x", etcdMember.ID), Endpoint: endpoint, DBSize: etcdStatus.DbSize, Version: etcdStatus.Version})
				}

			}

			response.Results = results

		}

		// deal response
		response.Result = fmt.Sprintf("'status' success (took %v)", utils.RoundDownDuration(time.Since(reqStart), minScaleDisplayUnit))
		return json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

type KeyValuePair struct {
	Key   string
	Value string
}

type GetClientResponse struct {
	Success bool
	Result  string
	//Results   []string
	KeyValues []KeyValuePair
}

func ClientGetHandle(c context.Context, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	reqStart := time.Now()
	isV2 := c.Value("isV2").(bool)

	switch r.Method {
	case http.MethodGet:
		response := GetClientResponse{Success: true}

		// translate parameters
		queries := r.URL.Query()

		timeoutParam := queries.Get("Timeout")
		timeout, err := time.ParseDuration(timeoutParam)
		if err != nil {
			timeout = 5
		}
		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
		defer timeoutCancelFn()

		keyParam := queries.Get("Key")
		key := template.HTMLEscapeString(keyParam)
		if keyParam == "" {
			response.Success = false
			response.Result = "Key required"
			return json.NewEncoder(w).Encode(response)
		}

		if isV2 {
			// translate parameters
			v2SortParam := queries.Get("Sort")

			v2QuorumParam := queries.Get("Quorum")

			v2Sort, err := strconv.ParseBool(v2SortParam)
			if err != nil {
				v2Sort = false
			}
			v2Quorum, err := strconv.ParseBool(v2QuorumParam)
			if err != nil {
				v2Quorum = false
			}

			// create opts
			v2Opts := v2.GetOptions{Sort: v2Sort, Quorum: v2Quorum}

			// call etcd
			v2Client, err := getV2Client(c)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}
			v2ClientApi := v2.NewKeysAPI(v2Client)
			etcdResponse, err := v2ClientApi.Get(timeoutCtx, key, &v2Opts)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}

			if etcdResponse.Node.Dir {
				response.Success = false
				response.Result = fmt.Sprintf("%s: is a directory", etcdResponse.Node.Key)
				return json.NewEncoder(w).Encode(response)
			}

			// deal return
			if etcdResponse.Node != nil {
				responseKvs := make([]KeyValuePair, 1)
				responseKvs[0] = KeyValuePair{etcdResponse.Node.Key, etcdResponse.Node.Value}
				response.KeyValues = responseKvs
			}

		} else {
			// translate parameters
			v3PrefixParam := queries.Get("Prefix")
			v3ConsistencyParam := queries.Get("Consistency")
			v3SortOrderParam := queries.Get("SortOrder")
			v3SortTargetParam := queries.Get("SortBy")
			v3LimitParam := queries.Get("Limit")
			v3FromKeyParam := queries.Get("FromKey")
			v3RevParam := queries.Get("Rev")
			v3KeysOnlyParam := queries.Get("KeysOnly")
			v3RangeParam := queries.Get("Range") // (fromKey|prefix)

			v3Prefix, err := strconv.ParseBool(v3PrefixParam)
			if err != nil {
				v3Prefix = false
			}
			v3FromKey, err := strconv.ParseBool(v3FromKeyParam)
			if err != nil {
				v3FromKey = false
			}

			if v3Prefix && v3FromKey {
				response.Success = false
				response.Result = `"Prefix" and "FromKey" cannot be set at the same time, choose one.`
				return json.NewEncoder(w).Encode(response)
			}

			// create opts
			var v3Opts []v3.OpOption

			if v3ConsistencyParam != "" {
				switch v3ConsistencyParam {
				case "s":
					v3Opts = append(v3Opts, v3.WithSerializable())
				case "l":
				default:
					response.Success = false
					response.Result = fmt.Sprintf("unknown Consistency flag %s", v3ConsistencyParam)
					return json.NewEncoder(w).Encode(response)
				}
			}

			if v3RangeParam != "" {
				v3Opts = append(v3Opts, v3.WithRange(v3RangeParam))
			}

			v3Limit, err := strconv.ParseInt(v3LimitParam, 10, 64)
			if err != nil {
				v3Limit = 0
			}
			v3Opts = append(v3Opts, v3.WithLimit(v3Limit))

			v3Rev, err := strconv.ParseInt(v3RevParam, 10, 64)
			if err != nil {
				v3Rev = 0
			}
			if v3Rev > 0 {
				v3Opts = append(v3Opts, v3.WithRev(v3Rev))
			}

			v3SortOrder := v3.SortNone
			v3SortOrderParam = strings.ToUpper(v3SortOrderParam)
			switch {
			case v3SortOrderParam == "ASCEND":
				v3SortOrder = v3.SortAscend
			case v3SortOrderParam == "DESCEND":
				v3SortOrder = v3.SortDescend
			case v3SortOrderParam == "":
				// nothing
			default:
				response.Success = false
				response.Result = fmt.Sprintf("bad sort order %v", v3SortOrderParam)
				return json.NewEncoder(w).Encode(response)
			}

			v3SortTarget := v3.SortByKey
			v3SortTargetParam = strings.ToUpper(v3SortTargetParam)
			switch {
			case v3SortTargetParam == "CREATE":
				v3SortTarget = v3.SortByCreateRevision
			case v3SortTargetParam == "KEY":
				v3SortTarget = v3.SortByKey
			case v3SortTargetParam == "MODIFY":
				v3SortTarget = v3.SortByModRevision
			case v3SortTargetParam == "VALUE":
				v3SortTarget = v3.SortByValue
			case v3SortTargetParam == "VERSION":
				v3SortTarget = v3.SortByVersion
			case v3SortTargetParam == "":
				// nothing
			default:
				response.Success = false
				response.Result = fmt.Sprintf("bad sort target %v", v3SortTargetParam)
				return json.NewEncoder(w).Encode(response)
			}

			v3Opts = append(v3Opts, v3.WithSort(v3SortTarget, v3SortOrder))

			if v3Prefix {
				if len(key) == 0 {
					key = "\x00"
					v3Opts = append(v3Opts, v3.WithFromKey())
				} else {
					v3Opts = append(v3Opts, v3.WithPrefix())
				}
			}

			if v3FromKey {
				if len(key) == 0 {
					key = "\x00"
				}
				v3Opts = append(v3Opts, v3.WithFromKey())
			}

			v3KeysOnly, err := strconv.ParseBool(v3KeysOnlyParam)
			if err != nil {
				v3KeysOnly = false
			}
			if v3KeysOnly {
				v3Opts = append(v3Opts, v3.WithKeysOnly())
			}

			// call etcd
			v3Client, err := getV3Client(c)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}
			etcdResponse, err := v3Client.Get(timeoutCtx, key, v3Opts...)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}

			// deal return
			if len(etcdResponse.Kvs) != 0 {
				responseKvs := make([]KeyValuePair, len(etcdResponse.Kvs))
				for idx := range etcdResponse.Kvs {
					responseKvs[idx] = KeyValuePair{string(etcdResponse.Kvs[idx].Key), string(etcdResponse.Kvs[idx].Value)}
				}
				response.KeyValues = responseKvs
			}

		}

		// deal response
		response.Result = fmt.Sprintf("'get' success (took %v)", utils.RoundDownDuration(time.Since(reqStart), minScaleDisplayUnit))
		return json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

type LsClientResponse struct {
	Success bool
	Result  string
	Results []string
}

func ClientV2LsHandle(c context.Context, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	reqStart := time.Now()
	isV2 := c.Value("isV2").(bool)

	if !isV2 {
		http.Error(w, "Method Not Allowed", 405)
		return nil
	}

	switch r.Method {
	case http.MethodGet:
		response := LsClientResponse{Success: true}

		// translate parameters
		queries := r.URL.Query()

		timeoutParam := queries.Get("Timeout")
		timeout, err := time.ParseDuration(timeoutParam)
		if err != nil {
			timeout = 5
		}
		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
		defer timeoutCancelFn()

		keyParam := queries.Get("Key")
		key := "/"
		if keyParam != "" {
			key = template.HTMLEscapeString(keyParam)
		}

		// translate parameters
		v2SortParam := queries.Get("Sort")
		v2QuorumParam := queries.Get("Quorum")
		v2RecursiveParam := queries.Get("Recursive")
		v2FillPathParam := queries.Get("FillPath")

		v2Sort, err := strconv.ParseBool(v2SortParam)
		if err != nil {
			v2Sort = false
		}
		v2Quorum, err := strconv.ParseBool(v2QuorumParam)
		if err != nil {
			v2Quorum = false
		}
		v2Recursive, err := strconv.ParseBool(v2RecursiveParam)
		if err != nil {
			v2Recursive = false
		}
		v2FillPath, err := strconv.ParseBool(v2FillPathParam)
		if err != nil {
			v2FillPath = false
		}

		// create opts
		v2Opts := v2.GetOptions{Sort: v2Sort, Quorum: v2Quorum, Recursive: v2Recursive}

		// call etcd
		v2Client, err := getV2Client(c)
		if err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}
		v2ClientApi := v2.NewKeysAPI(v2Client)
		etcdResponse, err := v2ClientApi.Get(timeoutCtx, key, &v2Opts)
		if err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}

		// deal return
		response.Results = displaySimple.V2SprintLsResponse(c, etcdResponse, v2FillPath)

		// deal response
		response.Result = fmt.Sprintf("'ls' success (took %v)", utils.RoundDownDuration(time.Since(reqStart), minScaleDisplayUnit))
		return json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

func translateLeaseId(arg string) (leadId v3.LeaseID, err error) {
	id, err := strconv.ParseInt(arg, 16, 64)
	if err != nil {
		return 0, err
	}
	return v3.LeaseID(id), nil
}

type LeaseClientRequest struct {
	Timeout time.Duration
	TTL     int64

	LeaseId            string
	LeaseKeepAliveOnce bool
}

type LeaseClientResponse struct {
	Success bool
	Result  string
	Results []string

	LeaseId string
	TTL     int64
}

func ClientV3LeaseHandle(c context.Context, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	reqStart := time.Now()
	isV2 := c.Value("isV2").(bool)

	if isV2 {
		http.Error(w, "Method Not Allowed", 405)
		return nil
	}

	switch r.Method {
	case http.MethodGet:
		response := LeaseClientResponse{Success: true}

		// translate parameters
		queries := r.URL.Query()

		timeoutParam := queries.Get("Timeout")
		timeout, err := time.ParseDuration(timeoutParam)
		if err != nil {
			timeout = 5
		}
		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
		defer timeoutCancelFn()

		leaseIdParam := queries.Get("LeaseId")
		//if leaseIdParam != "" {
		// time-to-live
		// translate parameters
		leaseId, err := translateLeaseId(template.HTMLEscapeString(leaseIdParam))
		if err != nil {
			response.Success = false
			response.Result = fmt.Sprintf("cannot translate LeaseId %s", leaseIdParam)
			return json.NewEncoder(w).Encode(response)
		}

		showKeysParam := queries.Get("ShowKeys")
		showKeys, err := strconv.ParseBool(showKeysParam)
		if err != nil {
			showKeys = false
		}

		// create opts
		var opts []v3.LeaseOption
		if showKeys {
			opts = append(opts, v3.WithAttachedKeys())
		}

		// call etcd
		v3Client, err := getV3Client(c)
		if err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}
		etcdResponse, err := v3Client.TimeToLive(timeoutCtx, leaseId, opts...)
		if err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}

		// deal response
		response.Result = fmt.Sprintf("'lease time-to-live' success (took %v)", utils.RoundDownDuration(time.Since(reqStart), minScaleDisplayUnit))
		response.Results = displaySimple.V3SprintLeaseTimeToLiveResponse(c, etcdResponse, showKeys)
		return json.NewEncoder(w).Encode(response)
		//} else {
		//	// ls
		//	// translate parameters
		//
		//	// create opts
		//	// call etcd
		//	v3Client := getV3Client(c)
		//	etcdResponse, err := v3Client.Leases(timeoutCtx)
		//	if err != nil {
		//		response.Success = false
		//		response.Result = err.Error()
		//		return json.NewEncoder(w).Encode(response)
		//	}
		//
		//	// deal response
		//	response.Result = fmt.Sprintf("'lease list' success (took %v)", utils.RoundDownDuration(time.Since(reqStart), minScaleDisplayUnit))
		//	response.Results = displaySimple.V3SprintLeasesResponse(c, etcdResponse)
		//	return json.NewEncoder(w).Encode(response)
		//}
	case http.MethodDelete:
		response := LeaseClientResponse{Success: true}

		// translate parameters
		queries := r.URL.Query()

		timeoutParam := queries.Get("Timeout")
		timeout, err := time.ParseDuration(timeoutParam)
		if err != nil {
			timeout = 5
		}
		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
		defer timeoutCancelFn()

		leaseIdParam := queries.Get("LeaseId")
		leaseId, err := translateLeaseId(template.HTMLEscapeString(leaseIdParam))
		if err != nil {
			response.Success = false
			response.Result = fmt.Sprintf("cannot translate LeaseId %s", leaseIdParam)
			return json.NewEncoder(w).Encode(response)
		}

		// call etcd
		v3Client, err := getV3Client(c)
		if err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}
		_, err = v3Client.Revoke(timeoutCtx, leaseId)
		if err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}

		// deal response
		response.Result = fmt.Sprintf("'lease revoke' success (took %v)", utils.RoundDownDuration(time.Since(reqStart), minScaleDisplayUnit))
		return json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		response := LeaseClientResponse{Success: true}

		// translate parameters
		request := LeaseClientRequest{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}

		timeout := request.Timeout
		if timeout == 0 {
			timeout = 5
		}
		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
		defer timeoutCancelFn()

		ttl := request.TTL
		if ttl == 0 {
			response.Success = false
			response.Result = "bad TTL 0"
			return json.NewEncoder(w).Encode(response)
		}

		// call etcd
		v3Client, err := getV3Client(c)
		if err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}
		etcdResponse, err := v3Client.Grant(timeoutCtx, ttl)
		if err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}

		// deal response
		response.TTL = etcdResponse.TTL
		response.LeaseId = fmt.Sprintf("%016x", etcdResponse.ID)

		response.Result = fmt.Sprintf("'lease grant' success (took %v)", utils.RoundDownDuration(time.Since(reqStart), minScaleDisplayUnit))
		return json.NewEncoder(w).Encode(response)

	case http.MethodPut:
		response := LeaseClientResponse{Success: true}

		// translate parameters
		request := LeaseClientRequest{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}

		timeout := request.Timeout
		if timeout == 0 {
			timeout = 5
		}
		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
		defer timeoutCancelFn()

		leaseId, err := translateLeaseId(template.HTMLEscapeString(request.LeaseId))
		if err != nil {
			response.Success = false
			response.Result = fmt.Sprintf("cannot translate LeaseId %s", request.LeaseId)
			return json.NewEncoder(w).Encode(response)
		}

		// call etcd
		v3Client, err := getV3Client(c)
		if err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}
		if request.LeaseKeepAliveOnce {
			_, err := v3Client.KeepAliveOnce(timeoutCtx, leaseId)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}

		} else {
			etcdResponse, err := v3Client.KeepAlive(timeoutCtx, leaseId)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}

			var results []string
			for resp := range etcdResponse {
				results = append(results, fmt.Sprintf("lease %016x keepalived with TTL(%ds).", resp.ID, resp.TTL))
			}
			response.Results = results
		}

		response.Result = fmt.Sprintf("'lease keepalive' success (took %v)", utils.RoundDownDuration(time.Since(reqStart), minScaleDisplayUnit))
		return json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

type SetClientRequest struct {
	Timeout time.Duration
	Key     string
	Value   string

	// v2
	TTL           int32
	SwapWithValue string
	SwapWithIndex int32

	// v3
	Lease       string
	PrevKV      bool
	IgnoreValue bool
	IgnoreLease bool
}

type SetClientResponse struct {
	Success bool
	Result  string
	Results []string
}

func ClientSetHandle(c context.Context, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	reqStart := time.Now()
	isV2 := c.Value("isV2").(bool)
	switch r.Method {
	case http.MethodPost:
		response := SetClientResponse{Success: true}

		// translate parameters
		request := SetClientRequest{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			response.Success = false
			response.Result = err.Error()
			return json.NewEncoder(w).Encode(response)
		}

		timeout := request.Timeout
		if timeout == 0 {
			timeout = 5
		}
		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
		defer timeoutCancelFn()

		key := request.Key
		if len(key) != 0 && key != "" {
			key = template.HTMLEscapeString(key)
		} else {
			response.Success = false
			response.Result = "Key required"
			return json.NewEncoder(w).Encode(response)
		}

		value := request.Value
		if len(value) != 0 && value != "" {
			value = template.HTMLEscapeString(value)
		} else {
			response.Success = false
			response.Result = "Value required"
			return json.NewEncoder(w).Encode(response)
		}

		if isV2 {
			// create opts
			v2Opts := v2.SetOptions{TTL: time.Duration(request.TTL) * time.Second, PrevIndex: uint64(request.SwapWithIndex), PrevValue: request.SwapWithValue}

			// call etcd
			v2Client, err := getV2Client(c)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}
			v2ClientApi := v2.NewKeysAPI(v2Client)
			etcdResponse, err := v2ClientApi.Set(timeoutCtx, key, value, &v2Opts)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}

			// deal return
			response.Results = displaySimple.V2SprintSetResponse(c, etcdResponse)

		} else {
			// translate parameters
			if request.Lease == "" {
				request.Lease = "0"
			}

			leaseId, err := strconv.ParseInt(request.Lease, 16, 64)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}

			// create opts
			var v3Opts []v3.OpOption
			if leaseId != 0 {
				v3Opts = append(v3Opts, v3.WithLease(v3.LeaseID(leaseId)))
			}

			if request.PrevKV {
				v3Opts = append(v3Opts, v3.WithPrevKV())
			}

			if request.IgnoreValue {
				v3Opts = append(v3Opts, v3.WithIgnoreValue())
			}

			if request.IgnoreLease {
				v3Opts = append(v3Opts, v3.WithIgnoreLease())
			}

			// call etcd
			v3Client, err := getV3Client(c)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}
			etcdResponse, err := v3Client.Put(timeoutCtx, key, value, v3Opts...)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}

			// deal return
			response.Results = displaySimple.V3SprintSetResponse(c, request.PrevKV, !request.IgnoreValue, etcdResponse)

		}

		// deal response
		response.Result = fmt.Sprintf("'set' success (took %v)", utils.RoundDownDuration(time.Since(reqStart), minScaleDisplayUnit))
		return json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

type RemoveClientResponse struct {
	Success bool
	Result  string
	Results []string
}

func ClientRemoveHandle(c context.Context, w http.ResponseWriter, r *http.Request) error {
	defer r.Body.Close()

	reqStart := time.Now()
	isV2 := c.Value("isV2").(bool)

	switch r.Method {
	case http.MethodDelete:
		response := RemoveClientResponse{Success: true}

		// translate parameters
		queries := r.URL.Query()

		timeoutParam := queries.Get("Timeout")
		timeout, err := time.ParseDuration(timeoutParam)
		if err != nil {
			timeout = 5
		}
		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
		defer timeoutCancelFn()

		keyParam := queries.Get("Key")
		key := template.HTMLEscapeString(keyParam)
		if keyParam == "" {
			response.Success = false
			response.Result = "Key required"
			return json.NewEncoder(w).Encode(response)
		}

		if isV2 {
			// translate parameters
			v2DirParam := queries.Get("Dir")
			v2RecursiveParam := queries.Get("Recursive")
			v2WithValueParam := queries.Get("WithValue")
			v2WithIndexParam := queries.Get("WithIndex")

			v2Dir, err := strconv.ParseBool(v2DirParam)
			if err != nil {
				v2Dir = false
			}
			v2Recursive, err := strconv.ParseBool(v2RecursiveParam)
			if err != nil {
				v2Recursive = false
			}

			v2WithIndex, err := strconv.ParseInt(v2WithIndexParam, 10, 32)
			if err != nil {
				v2WithIndex = 0
			}

			// create opts
			v2Opts := &v2.DeleteOptions{PrevIndex: uint64(v2WithIndex), PrevValue: v2WithValueParam, Dir: v2Dir, Recursive: v2Recursive}

			// call etcd
			v2Client, err := getV2Client(c)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}
			v2ClientApi := v2.NewKeysAPI(v2Client)
			etcdResponse, err := v2ClientApi.Delete(timeoutCtx, key, v2Opts)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}

			// deal return
			response.Results = displaySimple.V2SprintDelResponse(c, etcdResponse)

		} else {
			// translate parameters
			v3PrefixParam := queries.Get("Prefix")
			v3FromKeyParam := queries.Get("FromKey")
			v3PrevKVParam := queries.Get("PrevKV")
			v3RangeParam := queries.Get("Range") // (fromKey|prefix)

			v3Prefix, err := strconv.ParseBool(v3PrefixParam)
			if err != nil {
				v3Prefix = false
			}
			v3FromKey, err := strconv.ParseBool(v3FromKeyParam)
			if err != nil {
				v3FromKey = false
			}

			if v3Prefix && v3FromKey {
				response.Success = false
				response.Result = `"Prefix" and "FromKey" cannot be set at the same time, choose one.`
				return json.NewEncoder(w).Encode(response)
			}

			// create opts
			var v3Opts []v3.OpOption

			if v3RangeParam != "" {
				v3Opts = append(v3Opts, v3.WithRange(v3RangeParam))
			}

			if v3Prefix {
				if len(key) == 0 {
					key = "\x00"
					v3Opts = append(v3Opts, v3.WithFromKey())
				} else {
					v3Opts = append(v3Opts, v3.WithPrefix())
				}
			}

			v3PrevKV, err := strconv.ParseBool(v3PrevKVParam)
			if err != nil {
				v3PrevKV = false
			}
			if v3PrevKV {
				v3Opts = append(v3Opts, v3.WithPrevKV())
			}

			if v3FromKey {
				if len(key) == 0 {
					key = "\x00"
				}
				v3Opts = append(v3Opts, v3.WithFromKey())
			}

			// call etcd
			v3Client, err := getV3Client(c)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}
			etcdResponse, err := v3Client.Delete(timeoutCtx, key, v3Opts...)
			if err != nil {
				response.Success = false
				response.Result = err.Error()
				return json.NewEncoder(w).Encode(response)
			}

			// deal return
			response.Results = displaySimple.V3SprintDelResponse(c, v3PrevKV, false, etcdResponse)
		}

		// deal response
		response.Result = fmt.Sprintf("'remove' success (took %v)", utils.RoundDownDuration(time.Since(reqStart), minScaleDisplayUnit))
		return json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method Not Allowed", 405)
	}

	return nil
}

//func ClusterBackupHandle(c context.Context, w http.ResponseWriter, r *http.Request) error {
//	defer r.Body.Close()
//
//	reqStart := time.Now()
//	isV2 := c.Value("isV2").(bool)
//
//	switch r.Method {
//	case http.MethodGet:
//		response := RemoveClientResponse{Success: true}
//		defer func() {
//			if !response.Success {
//				log.Error(response.Result)
//			}
//		}()
//
//		// translate parameters
//		queries := r.URL.Query()
//
//		timeoutParam := queries.Get("Timeout")
//		timeout, err := time.ParseDuration(timeoutParam)
//		if err != nil {
//			timeout = 5
//		}
//		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
//		defer timeoutCancelFn()
//
//		downloadFileParam := queries.Get("DownloadFile")
//		downloadFile := template.HTMLEscapeString(downloadFileParam)
//		if downloadFileParam != "" {
//			// downlaod one file
//
//		} else {
//			// list
//		}
//
//	case http.MethodPost:
//		response := RemoveClientResponse{Success: true}
//		defer func() {
//			if !response.Success {
//				log.Error(response.Result)
//			}
//		}()
//
//		// backup
//	case http.MethodDelete:
//		response := RemoveClientResponse{Success: true}
//		defer func() {
//			if !response.Success {
//				log.Error(response.Result)
//			}
//		}()
//
//		// translate parameters
//		queries := r.URL.Query()
//
//		timeoutParam := queries.Get("Timeout")
//		timeout, err := time.ParseDuration(timeoutParam)
//		if err != nil {
//			timeout = 5
//		}
//		timeoutCtx, timeoutCancelFn := context.WithTimeout(c, timeout*time.Second)
//		defer timeoutCancelFn()
//
//		downloadFileParam := queries.Get("DownloadFile")
//		downloadFile := template.HTMLEscapeString(downloadFileParam)
//		if downloadFileParam != "" {
//			// downlaod one file
//
//		} else {
//			// list
//		}
//
//	default:
//		http.Error(w, "Method Not Allowed", 405)
//	}
//
//	return nil
//}
