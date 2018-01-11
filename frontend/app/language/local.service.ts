import { Injectable } from '@angular/core';
import { Language as ZhLanguage } from './zh/language';
import * as _ from 'lodash';

@Injectable()
export class LocalService {
	
	NOT_FOUND_PAGE = 'Sorry, this page does not exist!';

 	NAVIGATION_CLUSTER_STATUS = 'ClusterStatus';
 	NAVIGATION_CLUSTER_BACKUP = 'Backup';
 	NAVIGATION_CLIENT ='Client';

 	MEMBER_CARD_TITLE_ID = 'ID:';
 	MEMBER_CARD_TITLE_ENDPOINT = 'Endpoint:';
 	MEMBER_CARD_TITLE_STATE = 'State:';
 	MEMBER_CARD_TITLE_DBSIZE = 'DB Size:';
 	MEMBER_CARD_TITLE_VERSION = 'Version:';
 	MEMBER_CARD_TITLE_HASH = 'Hash:';

 	BACKUP_COLUMN_NAME = 'Name';
 	BACKUP_COLUMN_SIZE = 'Size';
 	BACKUP_COLUMN_CREATE_TIME = 'Create Time';
 	BACKUP_COLUMN_OPS = 'Operation';
 	BACKUP_OP_SUBMIT = 'Create Backup';
 	BACKUP_OP_DELETE = 'Delete';

 	CLIENT_OP_WRITE = 'Write';
 	CLIENT_OP_READ = 'Read';
 	CLIENT_OP_REMOVE = 'Remove';
 	CLIENT_OP_SUBMIT = 'Submit';
 	CLIENT_OP_CHECKBOX_PREFIX = 'prefix';
 	CLIENT_OP_CHECKBOX_IGNORE_VALUE = 'ignoreValue';

 	CLIENT_INPUT_PLACEHOLDER_KEY = 'Type your key...';
 	CLIENT_INPUT_PLACEHOLDER_VALUE = 'Type your value...';

 	CLIENT_LOG = '[ Log ]';
 	CLIENT_LOG_CLEAN = 'Clean';

	constructor(){
		if (navigator.language == "zh-CN") {
			_.assign(this, ZhLanguage);
		}
	}
}