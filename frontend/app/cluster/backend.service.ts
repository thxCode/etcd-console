import { Injectable } from '@angular/core';
import { Http, Response } from '@angular/http';
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
export class MemberStatus {
  Name: string;
  ID: string;
  Endpoint: string;

  IsLeader: boolean;
  State: string;

  DBSize: number;
  Version: string;

  circleCoord: Coordinate;
  txtCoord: Coordinate;

  constructor(
    name: string,
    id: string,
    endpoint: string,
    isLeader: boolean,
    dbSize: number,
    version: string,
    circleCoord: Coordinate,
    txtCoord: Coordinate
  ) {
    this.Name = name;
    this.ID = id;
    this.Endpoint = endpoint;

    this.IsLeader = isLeader;

    this.DBSize = dbSize;
    this.Version = version;

    if (isLeader) {
      this.State = 'Leader'
    } else {
      this.State = 'Follower'
    }

    this.circleCoord = circleCoord;
    this.txtCoord = txtCoord;
  }

  toString() {
    return this.Name;
  }
}

export class StatusCluster {
  Members: MemberStatus[]

  constructor(
    members: MemberStatus[]
  ) {
    this.Members = members;
  }
}

const DISTANCE = 1500;

@Injectable()
export class BackendService {
  private endpoints = {
    status: 'cluster/status',
    backup: 'cluster/backup'
  }

  clusterStatus: StatusCluster;

  constructor(private http: Http) {
    this.clusterStatus = new StatusCluster([]);
  }

  ///////////////////////////////////////////////////////
  private processHTTPResponseClusterStatus(res: Response) {
    let responseJson = res.json();
    if (responseJson.Success && responseJson.Results && responseJson.Results.length >= 1) {
      let radius = 3500;
      let members = [];
      let rad = 2 * Math.PI / responseJson.Results.length;
      _.forEach(responseJson.Results, (val: any, idx: number) => {
        let idxRad = rad * idx;

        let x = (1 + Math.cos(idxRad)) * radius + DISTANCE;
        let y = (1 - Math.sin(idxRad)) * radius + DISTANCE;

        members.push(new MemberStatus(val.Name, val.ID, val.Endpoint, val.IsLeader, val.DBSize, val.Version, new Coordinate(x, y, 300), new Coordinate(x+300, y+300, 200)))
      });

      return new StatusCluster(members);
    } else {
      return this.clusterStatus || {};
    }
  }

  private processHTTPErrorClusterStatus(error: any) {
    let errMsg = (error.message) ? error.message :
      error.status ? `${error.status} - ${error.statusText}` : 'Server error';
    console.error(errMsg);
    return Observable.throw(errMsg);
  }

  fetchClusterStatus(): Observable<StatusCluster> {
    return this.http.get(this.endpoints.status)
      .map(this.processHTTPResponseClusterStatus)
      .catch(this.processHTTPErrorClusterStatus);
  }
  ///////////////////////////////////////////////////////
}
