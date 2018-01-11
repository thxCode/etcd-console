import { Component, OnInit, AfterContentInit, OnDestroy } from '@angular/core';
import { SessionStorageService } from 'ngx-webstorage';
import { BackendService, MemberStatusResponse } from './backend.service';
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
  memberStatuses: MemberStatusResponse[];
  clusterStatusFetchHandler;

  constructor(
    private backendService: BackendService,
    private sessionStorage: SessionStorageService,
    public localService: LocalService,
  ) {
    this.selectedTab = 0;
  }

  ngOnInit() {
    this.memberStatuses = this.sessionStorage.retrieve('memberStatuses');
  }

  ngAfterContentInit() {
    this.connectCluster()
  }

  ngOnDestroy() {
    clearInterval(this.clusterStatusFetchHandler);
    this.sessionStorage.store('memberStatuses', this.memberStatuses);
  }

  connectCluster() {
    this.clusterStatusFetchHandler = setInterval(() => this.getMemberStatuses(), 2000);
  }

  selectTab(num: number) {
    this.selectedTab = num;
  }

  ///////////////////////////////////////////////////////
  getMemberStatuses() {
    this.responseError = null;

    this.backendService.fetchMemberStatuses().subscribe(
      memberStatusesResp => this.memberStatuses = memberStatusesResp,
      error => this.responseError = <any>error,
    );
  }
  ///////////////////////////////////////////////////////

}
