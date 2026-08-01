import { Routes } from '@angular/router';
import { canCreatePurchaseRequestGuard } from './procurement.guard';

export const procurementRoutes: Routes = [
  {
    path: '',
    loadComponent: () =>
      import('./pages/purchase-request-list/purchase-request-list').then(
        (module) => module.PurchaseRequestList,
      ),
    title: 'Phiếu mua sắm | DX-OS',
  },
  {
    path: 'new',
    canActivate: [canCreatePurchaseRequestGuard],
    loadComponent: () =>
      import('./pages/purchase-request-create/purchase-request-create').then(
        (module) => module.PurchaseRequestCreate,
      ),
    title: 'Tạo phiếu mua sắm | DX-OS',
  },
  {
    path: ':requestId/edit',
    canActivate: [canCreatePurchaseRequestGuard],
    loadComponent: () =>
      import('./pages/purchase-request-create/purchase-request-create').then(
        (module) => module.PurchaseRequestCreate,
      ),
    title: 'Sửa phiếu mua sắm | DX-OS',
  },
  {
    path: ':requestId',
    loadComponent: () =>
      import('./pages/purchase-request-detail/purchase-request-detail').then(
        (module) => module.PurchaseRequestDetail,
      ),
    title: 'Chi tiết phiếu mua sắm | DX-OS',
  },
];
