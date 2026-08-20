import { navigationForRoles, primaryRoleLabel } from './navigation.model';

function routesFor(roles: string[]): string[] {
  return navigationForRoles(roles).flatMap((group) => group.items.map((item) => item.route));
}

describe('navigationForRoles', () => {
  it('gives employees only their operational routes', () => {
    expect(routesFor(['employee'])).toEqual([
      '/dashboard',
      '/work-center',
      '/notifications',
      '/purchase-requests',
      '/operations',
      '/employee-guide',
    ]);
  });

  it('gives department managers approval work without finance or control modules', () => {
    expect(routesFor(['department_manager'])).toEqual([
      '/dashboard',
      '/work-center',
      '/approvals',
      '/approval-governance',
      '/notifications',
      '/purchase-requests',
      '/operations',
    ]);
  });

  it('does not give auditors mutating approval or administration routes', () => {
    const routes = routesFor(['auditor']);

    expect(routes).toContain('/audit');
    expect(routes).toContain('/budgets');
    expect(routes).toContain('/invoices');
    expect(routes).not.toContain('/approvals');
    expect(routes).not.toContain('/admin');
  });

  it('gives administrators system and control routes without procurement actions', () => {
    expect(routesFor(['dx_admin'])).toEqual([
      '/dashboard',
      '/approval-governance',
      '/notifications',
      '/sourcing',
      '/reports',
      '/audit',
      '/policies',
      '/ai-center',
      '/admin',
    ]);
  });

  it('preserves auth-only routes for accounts without a recognized business role', () => {
    expect(routesFor(['unknown'])).toEqual(['/dashboard', '/notifications']);
  });

  it('removes unauthorized groups instead of returning empty shells', () => {
    expect(navigationForRoles(['ai_operator']).map((group) => group.label)).toEqual([
      'Không gian làm việc',
      'Công việc',
      'Trí tuệ hỗ trợ',
    ]);
  });

  it('preserves group boundaries while combining permissions from multiple roles', () => {
    const groups = navigationForRoles(['employee', 'finance']);

    expect(groups.map((group) => [group.label, group.items.map((item) => item.route)])).toEqual([
      ['Không gian làm việc', ['/dashboard']],
      ['Công việc', ['/work-center', '/approvals', '/approval-governance', '/notifications']],
      ['Mua sắm', ['/purchase-requests', '/operations', '/suppliers', '/sourcing']],
      ['Tài chính', ['/budgets', '/invoices', '/reports']],
      ['Trí tuệ hỗ trợ', ['/ai-center']],
      ['Trợ giúp', ['/employee-guide']],
    ]);
  });

  it('derives each result from the current role collection without retaining prior access', () => {
    expect(routesFor(['finance'])).toContain('/approvals');
    expect(routesFor(['employee'])).not.toContain('/approvals');
  });

  it('uses the first recognized role for the role label', () => {
    expect(primaryRoleLabel(['finance', 'auditor'])).toBe('Tài chính');
    expect(primaryRoleLabel(['unknown'])).toBe('Đã xác thực');
    expect(primaryRoleLabel(['toString'])).toBe('Đã xác thực');
  });
});
