import { Injectable } from '@angular/core';
import { Language as ZhLanguage } from './zh/language';
import { Language as EnLanguage } from './en/language';

@Injectable()
export class LocalService {
	core: any;

	constructor(){
		if (navigator.language == "zh-CN") {
			this.core = ZhLanguage;
		} else {
			this.core = EnLanguage;
		}
	}
}