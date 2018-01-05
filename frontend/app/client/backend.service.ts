import { Injectable } from '@angular/core';
import { Http, Response, URLSearchParams, Headers, RequestOptions, RequestOptionsArgs, ResponseContentType} from '@angular/http';
import { Observable } from 'rxjs';
import 'rxjs/Rx';
import * as _ from 'lodash';

export class ProccessRequest {
  Action: string;

  constructor(action: string) {
    this.Action = action;
  }

  append(key: string, val: any): ProccessRequest {
    if (_.isBoolean(val)) {
      _.set(this, key, val);
      return this;
    }

    if (!_.isEmpty(val)) {
      _.set(this, key, val);
      return this;
    }

  }

}

class ClientRequest {

  bodyParams(): RequestOptionsArgs {
    let clone = _.cloneDeep(this);

    return new RequestOptions({
      responseType: ResponseContentType.Json,
      headers: new Headers({ 'Content-Type': 'application/json' }),
      body: JSON.stringify(clone)
    });
  }

  searchParams(): RequestOptionsArgs {
    let clone = _.cloneDeep(this);

    let params = new URLSearchParams();
    _.forEach(_.keys(clone), (key) => {
      if (clone[key]) {
        params.append(key, _.toString(clone[key]));
      }
    })

    return new RequestOptions({
      responseType: ResponseContentType.Json,
      search: params
    });
  }

}

export class GetClientRequest extends ClientRequest {
  Key: string;

  //v2
  Sort: boolean;
  Quorum: boolean;
  
  //v3
  Prefix: boolean;
  FromKey: boolean;
  Consistency: string;
  SortOrder: string;
  SortBy: string;
  Limit: number;
  Rev: number;
  KeysOnly: boolean;
  Range: string;

  constructor(
    key: string,

    //v2
    sort: boolean,
    quorum: boolean,

    //v3
    prefix: boolean,
    fromKey: boolean,
    consistency: string,
    sortOrder: string,
    sortBy: string,
    limit: number,
    rev: number,
    keysOnly: boolean,
    range: string,
  ){
    super();

    this.Key = key;

    this.Sort = sort;
    this.Quorum = quorum;

    this.Prefix = prefix;
    this.FromKey = fromKey;
    this.Consistency = consistency;
    this.SortOrder = sortOrder;
    this.SortBy = sortBy;
    this.Limit = limit;
    this.Rev = rev;
    this.KeysOnly = keysOnly;
    this.Range = range;
  }

  static newInstance(obj: Object) {
     return new GetClientRequest(
       obj['Key'], 
       obj['Sort'], 
       obj['Quorum'], 
       obj['Prefix'], 
       obj['FromKey'], 
       obj['Consistency'], 
       obj['SortOrder'],
       obj['SortBy'],
       obj['Limit'],
       obj['Rev'],
       obj['KeysOnly'],
       obj['Range']
     );
  }

}

export class SetClientRequest extends ClientRequest {
  Key: string;
  Value: string;

  //v2
  SwapWithIndex: number;
  SwapWithValue: string;

  //v3
  LeaseId: string;
  PrevKV: boolean;
  IgnoreValue: boolean;
  IgnoreLease: boolean;

  constructor(

    key: string,
    value: string,

    //v2
    swapWithIndex: number,
    swapWithValue: string,

    //v3
    leaseId: string,
    prevKV: boolean,
    ignoreValue: boolean,
    ignoreLease: boolean,
  ){
    super();

    this.Key = key;
    this.Value = value;

    this.SwapWithIndex = swapWithIndex;
    this.SwapWithValue = swapWithValue;

    this.LeaseId =leaseId;
    this.PrevKV = prevKV;
    this.IgnoreValue = ignoreValue;
    this.IgnoreLease = ignoreLease;
  }

  static newInstance(obj: Object) {
     return new SetClientRequest(
       obj['Key'], 
       obj['Value'], 
       obj['SwapWithIndex'], 
       obj['SwapWithValue'], 
       obj['LeaseId'], 
       obj['PrevKV'], 
       obj['IgnoreValue'],
       obj['IgnoreLease']
     );
  }

}

export class RemoveClientRequest extends ClientRequest {
  Key: string;

  //v2
  Dir: boolean;
  Recursive: boolean;
  WithValue: string;
  WithIndex: number;

  //v3
  Prefix: boolean;
  FromKey: boolean;
  PrevKV: boolean;
  Range: string;

  constructor(
    key: string,

    //v2
    dir: boolean,
    recursive: boolean,
    withValue: string,
    withIndex: number,

    //v3
    prefix: boolean,
    fromKey: boolean,
    prevKV: boolean,
    range: string,
  ){
    super();

    this.Key = key;

    this.Dir = dir;
    this.Recursive = recursive;
    this.WithValue = withValue;
    this.WithIndex = withIndex;

    this.Prefix =prefix;
    this.FromKey = fromKey;
    this.PrevKV = prevKV;
    this.Range = range;
  }

  static newInstance(obj: Object) {
     return new RemoveClientRequest(
       obj['Key'], 
       obj['Dir'], 
       obj['Recursive'], 
       obj['WithValue'], 
       obj['WithIndex'], 
       obj['Prefix'], 
       obj['FromKey'],
       obj['PrevKV'],
       obj['Range']
     );
  }

}

export class ProccessResponse {
  Level: number; // 0 - success, 1 - warn, 2 - error
  Results: string[];

  constructor(
    l: number,
    rss: string[]
  ) {
    this.Level = l;
    this.Results = rss;
  }
}

@Injectable()
export class BackendService {
  private endpoints = {
    read: 'client/get',
    write: 'client/set',
    remove: 'client/remove'
  }

  private jsonRequestOptions = new RequestOptions({
    responseType: ResponseContentType.Json,
    headers: new Headers({ 'Content-Type': 'application/json' })
  });

  constructor(private http: Http) {
  }

  process(request: ProccessRequest): Observable<ProccessResponse> {
    switch (request.Action) {
      case 'write':
        return this.set(SetClientRequest.newInstance(request));
      case 'remove':
        return this.remove(RemoveClientRequest.newInstance(request));
      default:
        return this.get(GetClientRequest.newInstance(request));
    }
  }

  private processHTTPErrorClient(error: any) {
    let errMsg = (error.message) ? error.message :
      error.status ? `${error.status} - ${error.statusText}` : 'Server error';
    console.error(errMsg);
    return Observable.throw(errMsg);
  }

  ///////////////////////////////////////////////////////
  private processHTTPResponseClientGet(res: Response) {
    let responseJson = res.json();
    if (responseJson.Success) {
      if (responseJson.KeyValues && responseJson.KeyValues.length > 0) {
        let rss = [responseJson.Result, '.'];

        _.forEach(responseJson.KeyValues, (kv) => {
          if (kv.Value) {
            rss.push(`|-- ${kv.Key} = ${kv.Value}`);
          } else {
            rss.push(`|-- ${kv.Key}`);
          }
        });

        return new ProccessResponse(0, rss);
      } else {
        return new ProccessResponse(1, ['cannot read anything']);
      }
    } else {
       return new ProccessResponse(2, [responseJson.Result ? responseJson.Result : 'unknown error']);
    }
  }

  private get(request: GetClientRequest): Observable<ProccessResponse> {
    return this.http.get(this.endpoints.read, request.searchParams())
      .map(this.processHTTPResponseClientGet)
      .catch(this.processHTTPErrorClient);
  }
  ///////////////////////////////////////////////////////

  ///////////////////////////////////////////////////////
  private processHTTPResponseClientSet(res: Response) {
    let responseJson = res.json();
    if (responseJson.Success) {
      let rss = [responseJson.Result];

      if (responseJson.Results && responseJson.Results.length > 1) {
        _.forEach(responseJson.Results, (rs, idx) => {
          if (rs) {
            rss.push(`${idx}: ${rs}`);
          }
        });
      }
      
      return new ProccessResponse(0, rss);
    } else {
      return new ProccessResponse(2, [responseJson.Result ? responseJson.Result : 'unknown error']);
    }
  }

  private set(request: SetClientRequest): Observable<ProccessResponse> {
    return this.http.post(this.endpoints.write, null, request.bodyParams())
      .map(this.processHTTPResponseClientSet)
      .catch(this.processHTTPErrorClient);
  }
  ///////////////////////////////////////////////////////

  ///////////////////////////////////////////////////////
  private processHTTPResponseClientRemove(res: Response) {
    let responseJson = res.json();
    if (responseJson.Success) {
      let rss = [responseJson.Result];

      if (responseJson.Results && responseJson.Results.length > 1) {
        _.forEach(responseJson.Results, (rs, idx) => {
          if (rs) {
            rss.push(`${idx}: ${rs}`);
          }
        });
      }
      
      return new ProccessResponse(0, rss);
    } else {
      return new ProccessResponse(2, [responseJson.Result ? responseJson.Result : 'unknown error']);
    }
  }

  private remove(request: RemoveClientRequest): Observable<ProccessResponse> {
    return this.http.delete(this.endpoints.remove, request.searchParams())
      .map(this.processHTTPResponseClientRemove)
      .catch(this.processHTTPErrorClient);
  }
  ///////////////////////////////////////////////////////
}
