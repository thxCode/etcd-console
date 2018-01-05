import { Component, OnInit, AfterContentInit, OnDestroy } from '@angular/core';
import { SessionStorageService } from 'ngx-webstorage';
import { BackendService, StatusCluster, MemberStatus } from './backend.service';
import { LocalService } from '../language/local.service';

@Component({
  selector: 'cluster-status',
  templateUrl: 'status.component.html',
  styleUrls: ['status.component.css'],
  providers: [BackendService, LocalService],
})
export class StatusComponent implements OnInit, AfterContentInit, OnDestroy {

  responseError: any;

  selectedTab: number;
  memberStatuses: MemberStatus[];
  clusterStatusFetchHandler;

  constructor(
    private backendService: BackendService,
    private sessionStorage: SessionStorageService,
    public localService: LocalService,
  ) {
    this.selectedTab = 0;
  }

  ngOnInit() {
    this.memberStatuses = this.sessionStorage.retrieve('clusterMemberStatuses');
  }

  ngAfterContentInit() {
    this.connectCluster()
  }

  ngOnDestroy() {
    clearInterval(this.clusterStatusFetchHandler);
    this.sessionStorage.store('clusterMemberStatuses', this.memberStatuses);
    return;
  }

  connectCluster() {
    this.clusterStatusFetchHandler = setInterval(() => this.getClusterStatus(), 1000);
  }

  selectTab(num: number) {
    this.selectedTab = num;
  }

  ///////////////////////////////////////////////////////
  private processClusterStatusResponse(resp: StatusCluster) {
    this.responseError = null;
    this.memberStatuses = resp.Members;
  }

  // getServerStatus fetches server status from backend.
  // memberStatus is true to get the status of all nodes.
  getClusterStatus() {
    let clusterStatusResult: StatusCluster;

    this.backendService.fetchClusterStatus().subscribe(
      clusterStatus => clusterStatusResult = clusterStatus,
      error => this.responseError = <any>error,
      () => this.processClusterStatusResponse(clusterStatusResult),
    );
  }
  ///////////////////////////////////////////////////////

}
