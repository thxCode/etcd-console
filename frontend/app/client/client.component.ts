import { Component, AfterViewChecked, AfterContentInit } from '@angular/core';
import { Observable } from 'rxjs';
import { BackendService, ProccessRequest, ProccessResponse } from './backend.service';
import { LocalService } from '../language/local.service';
import * as _ from 'lodash';
import 'iscroll';


export class LogLine {
  index: number;
  logLevel: number;
  prefix: string;
  text: string;

  constructor(index: number, logLevel: number, text: string) {
    this.index = index;
    this.logLevel = logLevel;

    let date = new Date();
    let yr = date.getFullYear();
    let mo = date.getMonth() + 1;
    let da = date.getDate();
    let timestamp = date.toTimeString().substring(0, 8);
    let moTxt = String(mo);
    if (moTxt.length === 1) {
      moTxt = '0' + moTxt;
    }
    let daTxt = String(da);
    if (daTxt.length === 1) {
      daTxt = '0' + daTxt;
    }
    let timePrefix = String(yr) + '-' + moTxt + '-' + daTxt + ' ' + timestamp;

    let logLevelTxt;
    switch (logLevel) {
      case 1:
        logLevelTxt = 'WARN';
        break;
      case 2:
        logLevelTxt = 'ERR ';
        break;
      default:
        logLevelTxt = 'INFO';
        break;
    }
    this.prefix = '[' + timePrefix + ' ' + logLevelTxt + ']';

    this.text = text;
  }
}

@Component({
  selector: 'client',
  templateUrl: 'client.component.html',
  styleUrls: ['client.component.css'],
  providers: [BackendService, LocalService],
})
export class ClientComponent implements AfterViewChecked, AfterContentInit {
  logBoxIScroll: IScroll;
  logBoxIScrollRefresh: boolean;

  logOutputLines: LogLine[];
  selectedTab: number;

  inputKey: string;
  inputValue: string;
  inputPrefix: boolean;

  debounceCleanLogs = _.debounce(() => {
    this.logOutputLines.length = 0;
  }, 200);
  debounceScrollToLogsBottom = _.debounce(() => {
    this.logBoxIScroll.refresh();
    this.logBoxIScroll.scrollToElement('.log-box_scroller_item:last-child', 1000);
  }, 200);

  constructor(
    private backendService: BackendService,
    public localService: LocalService,
  ) {
    this.logOutputLines = [];

    this.selectedTab = 0;
  }

  ngAfterContentInit() {
      this.logBoxIScroll = new IScroll('#wrapper', {
        scrollbars: true,
        mouseWheel: true,
        interactiveScrollbars: true,
        shrinkScrollbars: 'scale',
        fadeScrollbars: true
      });
  }

  ngAfterViewChecked() {
    if (this.logBoxIScrollRefresh) {
      this.logBoxIScrollRefresh = false;
      try {
        this.debounceScrollToLogsBottom();
      } catch (err) {}
    }
  }

  selectTab(num: number) {
    this.selectedTab = num;
  }

  sendLogLine(logLevel: number, txt: string) {
    this.logOutputLines.push(new LogLine(this.logOutputLines.length, logLevel, txt));
  }

  cleanLogs() {
    this.debounceCleanLogs();
  }

  // https://angular.io/docs/ts/latest/guide/template-syntax.html
  trackByLineIndex(index: number, line: LogLine) {
    return line.index;
  }

  ///////////////////////////////////////////////////////
  processClientRequest(act: string) {
    let clientRequest = new ProccessRequest(act);
    clientRequest.append('key', this.inputKey);
    clientRequest.append('value', this.inputValue);
    clientRequest.append('prefix', this.inputPrefix);

    this.backendService.process(clientRequest).subscribe(
      clientResponse => {
        for (let idx in clientResponse.results) {
          this.sendLogLine(clientResponse.level, clientResponse.results[idx]);
        }
        this.logBoxIScrollRefresh = true;
      },
      error => {
        this.sendLogLine(2, error)
        this.logBoxIScrollRefresh = true;
      },
    );
  }
  ///////////////////////////////////////////////////////
}
