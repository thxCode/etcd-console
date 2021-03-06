import { Routes, RouterModule } from '@angular/router';

import { StatusComponent as ClusterStatusComponent } from './cluster/status.component';
import { BackupComponent as ClusterBackupComponent } from './cluster/backup.component';
import { ClientComponent } from './client/client.component';

import { NotFoundComponent } from './not-found.component';

const appRoutes: Routes = [
    { path: '', redirectTo: '/cluster-status', pathMatch: 'full' },
    { path: 'cluster-status', component: ClusterStatusComponent },
    { path: 'cluster-backup', component: ClusterBackupComponent },
    { path: 'client', component: ClientComponent },
    { path: '**', component: NotFoundComponent },
];

export const routing = RouterModule.forRoot(appRoutes, {useHash: true});

export const routedComponents = [
    ClusterStatusComponent,
    ClusterBackupComponent,
    ClientComponent,
    NotFoundComponent,
];
