import { Injectable } from '@angular/core';
import { HttpClient, HttpParams, HttpErrorResponse } from '@angular/common/http';
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

  json() {
    return JSON.stringify(_.cloneDeep(this));
  }

  params() {
    let clone = _.cloneDeep(this);

    let params = {};
    _.forEach(_.keys(clone), (key) => {
      if (clone[key]) {
        params[key] = _.toString(clone[key]);
      }
    })

    return params;
  }
}

export class ReadClientRequest extends ClientRequest {
  key: string;

  //v2
  sort: boolean;
  quorum: boolean;
  
  //v3
  prefix: boolean;
  fromKey: boolean;
  consistency: string;
  sortOrder: string;
  sortTarget: string;
  limit: number;
  rev: number;
  keysOnly: boolean;
  range: string;

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
    sortTarget: string,
    limit: number,
    rev: number,
    keysOnly: boolean,
    range: string,
  ){
    super();

    this.key = key;

    this.sort = sort;
    this.quorum = quorum;

    this.prefix = prefix;
    this.fromKey = fromKey;
    this.consistency = consistency;
    this.sortOrder = sortOrder;
    this.sortTarget = sortTarget;
    this.limit = limit;
    this.rev = rev;
    this.keysOnly = keysOnly;
    this.range = range;
  }

  static newInstance(obj: Object) {
     return new ReadClientRequest(
       obj['key'], 
       obj['sort'], 
       obj['quorum'], 
       obj['prefix'], 
       obj['fromKey'], 
       obj['consistency'], 
       obj['sortOrder'],
       obj['sortTarget'],
       obj['limit'],
       obj['rev'],
       obj['keysOnly'],
       obj['range']
     );
  }

}

export class WriteClientRequest extends ClientRequest {
  key: string;
  value: string;

  //v2
  ttl: number;
  swapWithIndex: number;
  swapWithValue: string;

  //v3
  lease: string;
  prevKV: boolean;
  ignoreValue: boolean;
  ignoreLease: boolean;

  constructor(

    key: string,
    value: string,

    //v2
    ttl: number,
    swapWithIndex: number,
    swapWithValue: string,

    //v3
    lease: string,
    prevKV: boolean,
    ignoreValue: boolean,
    ignoreLease: boolean,
  ){
    super();

    this.key = key;
    this.value = value;

    this.ttl = ttl;
    this.swapWithIndex = swapWithIndex;
    this.swapWithValue = swapWithValue;

    this.lease = lease;
    this.prevKV = prevKV;
    this.ignoreValue = ignoreValue;
    this.ignoreLease = ignoreLease;
  }

  static newInstance(obj: Object) {
     return new WriteClientRequest(
       obj['key'], 
       obj['value'], 
       obj['ttl'], 
       obj['swapWithIndex'], 
       obj['swapWithValue'], 
       obj['lease'], 
       obj['prevKV'], 
       obj['ignoreValue'],
       obj['ignoreLease']
     );
  }

}

export class RemoveClientRequest extends ClientRequest {
  key: string;

  //v2
  dir: boolean;
  recursive: boolean;
  withValue: string;
  withIndex: number;

  //v3
  prefix: boolean;
  fromKey: boolean;
  prevKV: boolean;
  range: string;

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

    this.key = key;

    this.dir = dir;
    this.recursive = recursive;
    this.withValue = withValue;
    this.withIndex = withIndex;

    this.prefix = prefix;
    this.fromKey = fromKey;
    this.prevKV = prevKV;
    this.range = range;
  }

  static newInstance(obj: Object) {
     return new RemoveClientRequest(
       obj['key'], 
       obj['dir'], 
       obj['recursive'], 
       obj['withValue'], 
       obj['withIndex'], 
       obj['prefix'], 
       obj['fromKey'],
       obj['prevKV'],
       obj['range']
     );
  }

}

export class ProccessResponse {
  level: number; // 0 - success, 1 - warn, 2 - error
  results: string[];

  constructor(
    l: number,
    rss: string[]
  ) {
    this.level = l;
    this.results = rss;
  }
}

@Injectable()
export class BackendService {
  private endpoints = {
    read: '/api/v1/client/read',
    write: '/api/v1/client/write',
    remove: '/api/v1/client/remove'
  }

  constructor(private http: HttpClient) {
  }

  process(request: ProccessRequest): Observable<ProccessResponse> {
    switch (request.Action) {
      case 'write':
        return this.write(WriteClientRequest.newInstance(request));
      case 'remove':
        return this.remove(RemoveClientRequest.newInstance(request));
      default:
        return this.read(ReadClientRequest.newInstance(request));
    }
  }

  private processHTTPErrorClient(err: HttpErrorResponse) {
    if (err.error) {
      return Observable.throw(err.error);
    } else {
      let errMsg = (err.message) ? err.message :
        err.status ? `${err.status} - ${err.statusText}` : 'Server error';
      return Observable.throw(errMsg);
    }
  }

  ///////////////////////////////////////////////////////
  private processHTTPResponseClientRead(responseJson: any) {
    if (!_.isEmpty(responseJson.kvs)) {
      let rss = [responseJson.result, '.'];

      _.forEach(responseJson.kvs, (kv) => {
        if (kv.value) {
          rss.push(`|-- ${kv.key} = ${kv.value}`);
          rss.push(`\\---- [crev: ${kv.createRevision}, rev: ${kv.modRevision}, ver: ${kv.version}, lease: ${kv.lease}]`)
        } else {
          rss.push(`|-- ${kv.key}`);
          rss.push(`\\----[crev: ${kv.createRevision}, rev: ${kv.modRevision}, ver: ${kv.version}, lease: ${kv.lease}]`)
        }
      });

      return new ProccessResponse(0, rss);
    } else {
      return new ProccessResponse(1, ['cannot read anything']);
    }
  }

  private read(request: ReadClientRequest): Observable<ProccessResponse> {
    return this.http.get(this.endpoints.read, {
      responseType: 'json',
      params: request.params()
    })
      .map(this.processHTTPResponseClientRead)
      .catch(this.processHTTPErrorClient);
  }
  ///////////////////////////////////////////////////////

  ///////////////////////////////////////////////////////
  private processHTTPResponseClientWrite(responseJson: any) {
    let rss = [responseJson.result];

    if (!_.isEmpty(responseJson.kvs)) {
      rss.push('.');

      _.forEach(responseJson.kvs, (kv) => {
        if (kv.value) {
          rss.push(`|-- ${kv.key} = ${kv.value}`);
          rss.push(`\\---- [crev: ${kv.createRevision}, rev: ${kv.modRevision}, ver: ${kv.version}, lease: ${kv.lease}]`)
        } else {
          rss.push(`|-- ${kv.key}`);
          rss.push(`\\---- [crev: ${kv.createRevision}, rev: ${kv.modRevision}, ver: ${kv.version}, lease: ${kv.lease}]`)
        }
      });
    }

    return new ProccessResponse(0, rss);
  }

  private write(request: WriteClientRequest): Observable<ProccessResponse> {
    return this.http.post(this.endpoints.write, request.json(), {
      responseType: 'json'
    })
      .map(this.processHTTPResponseClientWrite)
      .catch(this.processHTTPErrorClient);
  }
  ///////////////////////////////////////////////////////

  ///////////////////////////////////////////////////////
  private processHTTPResponseClientRemove(responseJson: any) {
    let rss = [responseJson.result];

    if (!_.isEmpty(responseJson.kvs)) {
      rss.push('.');

      _.forEach(responseJson.kvs, (kv) => {
        if (kv.value) {
          rss.push(`|-- ${kv.key} = ${kv.value}`);
          rss.push(`\\---- [crev: ${kv.createRevision}, rev: ${kv.modRevision}, ver: ${kv.version}, lease: ${kv.lease}]`)
        } else {
          rss.push(`|-- ${kv.key}`);
          rss.push(`  \\- [crev: ${kv.createRevision}, rev: ${kv.modRevision}, ver: ${kv.version}, lease: ${kv.lease}]`)
        }
      });
    }
    
    return new ProccessResponse(0, rss);
  }

  private remove(request: RemoveClientRequest): Observable<ProccessResponse> {
    return this.http.delete(this.endpoints.remove, {
      responseType: 'json',
      params: request.params()
    })
      .map(this.processHTTPResponseClientRemove)
      .catch(this.processHTTPErrorClient);
  }
  ///////////////////////////////////////////////////////
}
