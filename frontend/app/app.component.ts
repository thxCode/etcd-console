import { Component } from '@angular/core';
import { LocalService } from './language/local.service';

@Component({
  selector: 'app',
  templateUrl: 'app.component.html',
  styleUrls: ['app.component.css'],
  providers: [LocalService],
})
export class AppComponent {
	constructor(
		public localService: LocalService,
	){
	}
}
