export interface AdminOverview {
  organizationName: string;
  activeUsers: number;
  inactiveUsers: number;
  activeDepartments: number;
  openRequests: number;
  pendingNotifications: number;
  deadNotifications: number;
}

export interface AdminUser {
  id: string;
  username: string;
  email: string;
  displayName: string;
  departmentId: string;
  departmentName: string;
  active: boolean;
  version: number;
  updatedAt: string;
}

export interface AdminDepartment {
  id: string;
  code: string;
  name: string;
  costCenter: string;
  parentId?: string;
  active: boolean;
  version: number;
}

export interface AdminCenterModel {
  overview: AdminOverview;
  users: AdminUser[];
  departments: AdminDepartment[];
  roleNotice: string;
}

export interface UpdateAdminUser {
  displayName: string;
  email: string;
  departmentId: string;
  active: boolean;
  expectedVersion: number;
}

export interface SaveAdminDepartment {
  code: string;
  name: string;
  costCenter: string;
  parentId: string;
  active: boolean;
  expectedVersion: number;
}
