import { Routes } from '@angular/router';
import { authGuard } from './core/auth/auth.guard';
import { canAccessEmployeeGuideGuard } from './features/dashboard/employee-guide.guard';
import {
  canAccessBudgetManagementGuard,
  canAccessInvoicesGuard,
  canAccessPoliciesGuard,
  canAccessAuditGuard,
  canAccessOperationsGuard,
  canAccessReportsGuard,
  canAccessSupplierDirectoryGuard,
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
    path: 'notifications',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/dashboard/pages/notification-center').then(
        (module) => module.NotificationCenter,
      ),
    title: 'Thông báo | DX-OS',
  },
  {
    path: 'employee-guide',
    canActivate: [authGuard, canAccessEmployeeGuideGuard],
    loadComponent: () =>
      import('./features/dashboard/pages/employee-guide').then((module) => module.EmployeeGuide),
    title: 'Hướng dẫn nhân viên | DX-OS',
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
    path: 'work-center',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/procurement/pages/work-center/work-center').then(
        (module) => module.WorkCenter,
      ),
    title: 'Việc của tôi | DX-OS',
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
    path: 'invoices',
    canActivate: [authGuard, canAccessInvoicesGuard],
    loadComponent: () =>
      import('./features/procurement/pages/invoice-board/invoice-board').then(
        (module) => module.InvoiceBoardPage,
      ),
    title: 'Hóa đơn và thanh toán | DX-OS',
  },
  {
    path: 'policies',
    canActivate: [authGuard, canAccessPoliciesGuard],
    loadComponent: () =>
      import('./features/procurement/pages/policy-center/policy-center').then(
        (module) => module.PolicyCenterPage,
      ),
    title: 'Chính sách vận hành | DX-OS',
  },
  {
    path: 'suppliers',
    canActivate: [authGuard, canAccessSupplierDirectoryGuard],
    loadComponent: () =>
      import('./features/procurement/pages/supplier-directory/supplier-directory').then(
        (module) => module.SupplierDirectory,
      ),
    title: 'Nhà cung cấp | DX-OS',
  },
  {
    path: 'operations',
    canActivate: [authGuard, canAccessOperationsGuard],
    loadComponent: () =>
      import('./features/procurement/pages/operations-board/operations-board').then(
        (module) => module.OperationsBoard,
      ),
    title: 'Đặt hàng và giao nhận | DX-OS',
  },
  {
    path: 'audit',
    canActivate: [authGuard, canAccessAuditGuard],
    loadComponent: () =>
      import('./features/reporting/pages/audit-center/audit-center').then(
        (module) => module.AuditCenter,
      ),
    title: 'Trung tâm kiểm toán | DX-OS',
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
