import { Injectable } from '@angular/core';
import { Http, Response, URLSearchParams, Headers, RequestOptions, RequestOptionsArgs, ResponseContentType } from '@angular/http';
import { Observable } from 'rxjs';
import 'rxjs/Rx';
import * as _ from 'lodash';

export class Coordinate {
  x: number;
  y: number;
  r: number;

  constructor(
    x: number, 
    y: number,
    r: number
  ) {
    this.x = x;
    this.y = y;
    this.r = r;
  }
}

// MemberStatus defines etcd node status.
export class MemberStatusResponse {
  name: string;
  id: string;
  endpoint: string;

  isLeader: boolean;
  isHealth: boolean;
  isConnected: boolean;

  dbSize: string;
  version: string;

  state: string;
  circleCoord: Coordinate;
  txtCoord: Coordinate;

  constructor(
    name: string,
    id: string,
    endpoint: string,
    isLeader: boolean,
    isHealth: boolean,
    isConnected: boolean,
    dbSize: number,
    version: string,
    circleCoord: Coordinate,
    txtCoord: Coordinate
  ) {
    this.name = name;
    this.id = id;
    this.endpoint = endpoint;

    this.isLeader = isLeader;
    this.isHealth = isHealth;
    this.isConnected = isConnected;
    
    this.dbSize = _.toString(dbSize);
    this.version = version;

    if (isConnected) {
      if (isHealth) {
        if (isLeader) {
          this.state = 'Leader'
        } else {
          this.state = 'Follower'
        }
      } else {
        this.state = 'Stopped'
      }
    } else {
       this.state = 'Losted'
    }
    this.circleCoord = circleCoord;
    this.txtCoord = txtCoord;
  }

}

export class BackupResponse {
  name: string;
  size: string;
  createTime: string;

  constructor(
    name: string,
    size: number,
    createTime: string
  ) {
    this.name = name;
    this.size = _.toString(size);
    this.createTime = createTime;
  }
}

const DISTANCE = 1500;

@Injectable()
export class BackendService {
  private endpoints = {
    status: '/api/v1/cluster/status',
    backup: '/api/v1/cluster/backup'
  }

  memberStatuses: MemberStatusResponse[];
  backups: BackupResponse[];

  constructor(private http: Http) {
    this.memberStatuses = [];
    this.backups = [];
  }

  private processHTTPErrorCluster(error: any) {
    let errMsg = (error.message) ? error.message :
      error.status ? `${error.status} - ${error.statusText}` : 'Server error';
    return Observable.throw(errMsg);
  }

  ///////////////////////////////////////////////////////
  private processHTTPResponseMemberStatusesRead(res: Response) {
    let responseJson = res.json();

    if (!_.isEmpty(responseJson.members)) {
      let radius = 3500;
      let rad = 2 * Math.PI / responseJson.members.length;

      this.memberStatuses.length = 0;
      _.forEach(responseJson.members, (val: any, idx: number) => {
        let idxRad = rad * idx;

        let x = (1 + Math.cos(idxRad)) * radius + DISTANCE;
        let y = (1 - Math.sin(idxRad)) * radius + DISTANCE;

        this.memberStatuses.push(new MemberStatusResponse(val.name, val.id, val.endpoint, val.leader, val.health, val.connected, val.dbSize, val.version, new Coordinate(x, y, 300), new Coordinate(x+300, y+300, 200)))
      });
    }

    return this.memberStatuses;
  }

  fetchMemberStatuses(): Observable<MemberStatusResponse[]> {
    return this.http.get(
        this.endpoints.status,
        new RequestOptions({
          responseType: ResponseContentType.Json
        })
      )
      .map(this.processHTTPResponseMemberStatusesRead.bind(this))
      .catch(this.processHTTPErrorCluster);
  }
  ///////////////////////////////////////////////////////

  ///////////////////////////////////////////////////////
  private processHTTPResponseClusterBackupRead(res: Response) {
    let responseJson = res.json();

    this.backups.length = 0;
    if (!_.isEmpty(responseJson.backups)) {
      _.forEach(responseJson.backups, (backup: any) => {
          this.backups.push(new BackupResponse(backup.name, backup.size, backup.createTime))
      });
    }

    return this.backups;
  }

  fetchClusterBackups(): Observable<BackupResponse[]> {
    return this.http.get(this.endpoints.backup)
      .map(this.processHTTPResponseClusterBackupRead.bind(this))
      .catch(this.processHTTPErrorCluster);
  }
  ///////////////////////////////////////////////////////

  ///////////////////////////////////////////////////////
  removeClusterBackup(name: string): Observable<boolean> {
    let params = new URLSearchParams();
    params.append("name", name);

    return this.http.delete(
        this.endpoints.backup, 
        new RequestOptions({
          responseType: ResponseContentType.Json,
          search: params,
        })
      )
      .catch(this.processHTTPErrorCluster);
  }
  ///////////////////////////////////////////////////////

  ///////////////////////////////////////////////////////
  private processHTTPResponseClusterBackupCreate(res: Response) {
    let responseJson = res.json();

    if (!_.isEmpty(responseJson.backups)) {
      let backup = responseJson.backups[0];

      return new BackupResponse(backup.name, backup.size, backup.createTime)
    } else {
      return Observable.throw("backup error");
    }
  }

  createClusterBackup(): Observable<BackupResponse> {
    return this.http.post(
        this.endpoints.backup, 
        null,
        new RequestOptions({
          responseType: ResponseContentType.Json
        })
      )
      .map(this.processHTTPResponseClusterBackupCreate)
      .catch(this.processHTTPErrorCluster);
  }
  ///////////////////////////////////////////////////////

}
