import { Routes } from '@angular/router';
import { authGuard } from './core/auth/auth.guard';
import {
  canAccessBudgetManagementGuard,
  canAccessReportsGuard,
  canReviewPurchaseRequestsGuard,
} from './features/procurement/procurement.guard';

export const routes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    redirectTo: 'dashboard',
  },
  {
    path: 'dashboard',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/dashboard/pages/dashboard').then((module) => module.Dashboard),
    title: 'Tổng quan | DX-OS',
  },
  {
    path: 'approvals',
    canActivate: [authGuard, canReviewPurchaseRequestsGuard],
    loadComponent: () =>
      import('./features/procurement/pages/approval-inbox/approval-inbox').then(
        (module) => module.ApprovalInbox,
      ),
    title: 'Chờ phê duyệt | DX-OS',
  },
  {
    path: 'budgets',
    canActivate: [authGuard, canAccessBudgetManagementGuard],
    loadComponent: () =>
      import('./features/procurement/pages/budget-dashboard/budget-dashboard').then(
        (module) => module.BudgetDashboardPage,
      ),
    title: 'Ngân sách | DX-OS',
  },
  {
    path: 'reports',
    canActivate: [authGuard, canAccessReportsGuard],
    loadComponent: () =>
      import('./features/reporting/pages/report-dashboard/report-dashboard').then(
        (module) => module.ReportDashboardPage,
      ),
    title: 'Báo cáo vận hành | DX-OS',
  },
  {
    path: 'purchase-requests',
    canActivate: [authGuard],
    loadChildren: () =>
      import('./features/procurement/procurement.routes').then(
        (module) => module.procurementRoutes,
      ),
  },
  {
    path: '**',
    redirectTo: 'dashboard',
  },
];
