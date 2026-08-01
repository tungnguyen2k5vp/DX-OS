import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../../core/auth/auth.service';

export const canCreatePurchaseRequestGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const roles = auth.roles();
  if (roles.includes('employee') || roles.includes('department_manager')) {
    return true;
  }
  return inject(Router).createUrlTree(['/purchase-requests']);
};

export const canReviewPurchaseRequestsGuard: CanActivateFn = () => {
  const roles = inject(AuthService).roles();
  if (roles.includes('department_manager') || roles.includes('finance')) {
    return true;
  }
  return inject(Router).createUrlTree(['/purchase-requests']);
};

export const canAccessBudgetManagementGuard: CanActivateFn = () => {
  const roles = inject(AuthService).roles();
  if (roles.includes('finance') || roles.includes('auditor')) {
    return true;
  }
  return inject(Router).createUrlTree(['/dashboard']);
};

export const canAccessReportsGuard: CanActivateFn = () => {
  const roles = inject(AuthService).roles();
  if (roles.includes('finance') || roles.includes('auditor') || roles.includes('dx_admin')) {
    return true;
  }
  return inject(Router).createUrlTree(['/dashboard']);
};
