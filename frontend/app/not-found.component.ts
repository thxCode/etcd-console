import { Component } from '@angular/core';
import { LocalService } from './language/local.service';

@Component({
	selector: 'app-not-found',
	templateUrl: 'not-found.component.html',
	styleUrls: ['not-found.component.css'],
	providers: [LocalService],
})
export class NotFoundComponent {
	constructor(public localService: LocalService,){
	}
}
