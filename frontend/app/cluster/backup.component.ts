import { Component, ViewChild, AfterViewInit } from '@angular/core';
import { MatTableDataSource } from '@angular/material';
import { BackendService, BackupResponse } from './backend.service';
import { LocalService } from '../language/local.service';
import * as _ from 'lodash';

@Component({
  selector: 'cluster-backup',
  templateUrl: 'backup.component.html',
  styleUrls: ['backup.component.css'],
  providers: [BackendService, LocalService],
})
export class BackupComponent implements AfterViewInit {
  responseError: any;

  displayedColumns = ['name', 'size', 'createTime', 'ops'];
  dataSource = new MatTableDataSource();
  isLoadingResults = true;

  constructor(
    private backendService: BackendService,
    public localService: LocalService,
  ) {
  }

  ngAfterViewInit() {
    this.getBackups();
  }

  ///////////////////////////////////////////////////////
  getBackups() {
    this.responseError = null;
    this.isLoadingResults = true;

    this.backendService.fetchClusterBackups().subscribe(
      backupsResp => this.dataSource.data = backupsResp,
      error => {
        this.isLoadingResults = false;
        this.responseError = <any>error;
      },
      () => {
        window.setTimeout(() => {
          this.isLoadingResults = false;
        }, 500)
      },
    );
  }
  ///////////////////////////////////////////////////////

  ///////////////////////////////////////////////////////
  createBackup() {
    this.responseError = null;
    this.isLoadingResults = true;

    this.backendService.createClusterBackup().subscribe(
      backupResp => {
        if (!_.isEmpty(backupResp)) {
          this.getBackups()
        }
      },
      error => {
        this.responseError = <any>error;
        this.isLoadingResults = false;
      },
    );
  }
  ///////////////////////////////////////////////////////

  ///////////////////////////////////////////////////////
  deleteBackup(name: string) {
    this.responseError = null;
    this.isLoadingResults = true;

    this.backendService.removeClusterBackup(_.unescape(name)).subscribe(
      backupResp => {
        if (backupResp) {
          this.getBackups()
        }
      },
      error => {
        this.responseError = <any>error;
        this.isLoadingResults = false;
      },
    );
  }
  ///////////////////////////////////////////////////////
}
