export type AppRole =
  | 'employee'
  | 'department_manager'
  | 'finance'
  | 'auditor'
  | 'dx_admin'
  | 'ai_operator';

export interface NavigationItem {
  readonly label: string;
  readonly shortLabel: string;
  readonly route: string;
  readonly icon: NavigationIcon;
  readonly roles: readonly AppRole[];
  readonly availableToAuthenticatedUser?: boolean;
}

export interface NavigationGroup {
  readonly label: string;
  readonly items: readonly NavigationItem[];
}

export type NavigationIcon =
  | 'home'
  | 'tasks'
  | 'approval'
  | 'notifications'
  | 'requests'
  | 'operations'
  | 'suppliers'
  | 'budget'
  | 'invoices'
  | 'reports'
  | 'audit'
  | 'policies'
  | 'ai'
  | 'admin'
  | 'help';

const allRoles: readonly AppRole[] = [
  'employee',
  'department_manager',
  'finance',
  'auditor',
  'dx_admin',
  'ai_operator',
];

const procurementRoles: readonly AppRole[] = [
  'employee',
  'department_manager',
  'finance',
  'auditor',
];

const navigationGroups: readonly NavigationGroup[] = [
  {
    label: 'Không gian làm việc',
    items: [item('Tổng quan', 'Tổng quan', '/dashboard', 'home', [], true)],
  },
  {
    label: 'Công việc',
    items: [
      item('Việc của tôi', 'Công việc', '/work-center', 'tasks', procurementRoles),
      item(
        'Phê duyệt',
        'Phê duyệt',
        '/approvals',
        'approval',
        ['department_manager', 'finance'],
      ),
      item(
        'Ủy quyền và quy tắc',
        'Quy tắc',
        '/approval-governance',
        'approval',
        ['department_manager', 'finance', 'auditor', 'dx_admin'],
      ),
      item('Thông báo', 'Thông báo', '/notifications', 'notifications', [], true),
    ],
  },
  {
    label: 'Mua sắm',
    items: [
      item('Phiếu mua sắm', 'Mua sắm', '/purchase-requests', 'requests', procurementRoles),
      item('Đặt hàng & giao nhận', 'Giao nhận', '/operations', 'operations', procurementRoles),
      item('Nhà cung cấp', 'Nhà cung cấp', '/suppliers', 'suppliers', ['finance', 'auditor']),
      item('So sánh báo giá', 'Báo giá', '/sourcing', 'suppliers', ['finance', 'auditor', 'dx_admin']),
    ],
  },
  {
    label: 'Tài chính',
    items: [
      item('Ngân sách', 'Ngân sách', '/budgets', 'budget', ['finance', 'auditor']),
      item('Hóa đơn & thanh toán', 'Hóa đơn', '/invoices', 'invoices', ['finance', 'auditor']),
      item('Báo cáo', 'Báo cáo', '/reports', 'reports', ['finance', 'auditor', 'dx_admin']),
    ],
  },
  {
    label: 'Kiểm soát',
    items: [
      item('Kiểm toán', 'Kiểm toán', '/audit', 'audit', ['auditor', 'dx_admin']),
      item('Chính sách', 'Chính sách', '/policies', 'policies', ['auditor', 'dx_admin']),
    ],
  },
  {
    label: 'Trí tuệ hỗ trợ',
    items: [
      item(
        'Khuyến nghị',
        'Khuyến nghị',
        '/ai-center',
        'ai',
        ['ai_operator', 'dx_admin', 'finance', 'auditor'],
      ),
    ],
  },
  {
    label: 'Hệ thống',
    items: [item('Quản trị', 'Quản trị', '/admin', 'admin', ['dx_admin'])],
  },
  {
    label: 'Trợ giúp',
    items: [item('Hướng dẫn nhân viên', 'Hướng dẫn', '/employee-guide', 'help', ['employee'])],
  },
];

const roleLabels: Readonly<Record<AppRole, string>> = {
  employee: 'Nhân viên',
  department_manager: 'Trưởng bộ phận',
  finance: 'Tài chính',
  auditor: 'Kiểm toán',
  dx_admin: 'Quản trị DX-OS',
  ai_operator: 'Điều phối AI',
};

export function navigationForRoles(roles: readonly string[]): NavigationGroup[] {
  const granted = new Set(roles);

  return navigationGroups
    .map((group) => ({
      ...group,
      items: group.items.filter(
        (navigationItem) =>
          navigationItem.availableToAuthenticatedUser ||
          navigationItem.roles.some((role) => granted.has(role)),
      ),
    }))
    .filter((group) => group.items.length > 0);
}

export function primaryRoleLabel(roles: readonly string[]): string {
  const role = roles.find(
    (candidate): candidate is AppRole => Object.prototype.hasOwnProperty.call(roleLabels, candidate),
  );
  return role ? roleLabels[role] : 'Đã xác thực';
}

function item(
  label: string,
  shortLabel: string,
  route: string,
  icon: NavigationIcon,
  roles: readonly AppRole[],
  availableToAuthenticatedUser = false,
): NavigationItem {
  return { label, shortLabel, route, icon, roles, availableToAuthenticatedUser };
}
