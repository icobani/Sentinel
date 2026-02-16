import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    redirectTo: '/dashboard',
    pathMatch: 'full'
  },
  {
    path: 'dashboard',
    loadComponent: () => import('./features/dashboard/dashboard.component').then(m => m.DashboardComponent),
    data: { animation: 'DashboardPage' }
  },
  {
    path: 'setup',
    loadComponent: () => import('./features/setup/setup.component').then(m => m.SetupComponent),
    data: { animation: 'SetupPage' },
    children: [
      {
        path: '',
        loadComponent: () => import('./features/setup/watcher-list/watcher-list.component').then(m => m.WatcherListComponent)
      },
      {
        path: 'new',
        loadComponent: () => import('./features/setup/watcher-form/watcher-form.component').then(m => m.WatcherFormComponent)
      },
      {
        path: ':id',
        loadComponent: () => import('./features/setup/watcher-detail/watcher-detail.component').then(m => m.WatcherDetailComponent)
      },
      {
        path: ':id/edit',
        loadComponent: () => import('./features/setup/watcher-form/watcher-form.component').then(m => m.WatcherFormComponent)
      }
    ]
  },
  {
    path: 'logs',
    loadComponent: () => import('./features/logs/logs.component').then(m => m.LogsComponent),
    data: { animation: 'LogsPage' }
  },
  {
    path: '**',
    redirectTo: '/dashboard'
  }
];
