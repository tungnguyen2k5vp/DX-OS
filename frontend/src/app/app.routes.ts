import { Routes } from '@angular/router';
import { authGuard } from './core/auth/auth.guard';
import { canAccessEmployeeGuideGuard } from './features/dashboard/employee-guide.guard';
import {
  canAccessBudgetManagementGuard,
  canAccessInvoicesGuard,
  canAccessPoliciesGuard,
  canAccessProcurementGuard,
  canAccessAuditGuard,
  canAccessAdminGuard,
  canAccessAIGuard,
  canAccessOperationsGuard,
  canAccessReportsGuard,
  canAccessSupplierDirectoryGuard,
  canReviewPurchaseRequestsGuard,
  canAccessApprovalGovernanceGuard,
  canAccessSourcingGuard,
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
    canActivate: [authGuard, canAccessProcurementGuard],
    loadComponent: () =>
      import('./features/procurement/pages/work-center/work-center').then(
        (module) => module.WorkCenter,
      ),
    title: 'Việc của tôi | DX-OS',
  },
  {
    path: 'approval-governance',
    canActivate: [authGuard, canAccessApprovalGovernanceGuard],
    loadComponent: () =>
      import('./features/procurement/pages/approval-governance/approval-governance').then(
        (module) => module.ApprovalGovernancePage,
      ),
    title: 'Ủy quyền và quy tắc phê duyệt | DX-OS',
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
    path: 'sourcing',
    canActivate: [authGuard, canAccessSourcingGuard],
    loadComponent: () =>
      import('./features/procurement/pages/sourcing-board/sourcing-board').then(
        (module) => module.SourcingBoardPage,
      ),
    title: 'So sánh báo giá | DX-OS',
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
    path: 'admin',
    canActivate: [authGuard, canAccessAdminGuard],
    loadComponent: () =>
      import('./features/admin/pages/admin-center/admin-center').then(
        (module) => module.AdminCenterPage,
      ),
    title: 'Quản trị hệ thống | DX-OS',
  },
  {
    path: 'ai-assistant',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./features/ai/pages/assistant/assistant').then((module) => module.AIAssistantPage),
    title: 'Trợ lý AI nội bộ | DX-OS',
  },
  {
    path: 'ai-center',
    canActivate: [authGuard, canAccessAIGuard],
    loadComponent: () =>
      import('./features/ai/pages/recommendation-center/recommendation-center').then(
        (module) => module.RecommendationCenterPage,
      ),
    title: 'Trung tâm khuyến nghị | DX-OS',
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
    canActivate: [authGuard, canAccessProcurementGuard],
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
