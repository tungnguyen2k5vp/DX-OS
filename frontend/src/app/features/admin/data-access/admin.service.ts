import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { APP_CONFIG } from '../../../core/config/app-config';
import {
  AdminCenterModel,
  AdminDepartment,
  AdminUser,
  SaveAdminDepartment,
  UpdateAdminUser,
} from './admin.models';

@Injectable({ providedIn: 'root' })
export class AdminService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${inject(APP_CONFIG).apiBaseUrl}/api/v1/admin`;

  center(): Observable<AdminCenterModel> {
    return this.http.get<AdminCenterModel>(`${this.baseUrl}/center`);
  }

  updateUser(userId: string, input: UpdateAdminUser): Observable<AdminUser> {
    return this.http.patch<AdminUser>(`${this.baseUrl}/users/${encodeURIComponent(userId)}`, input);
  }

  createDepartment(input: SaveAdminDepartment): Observable<AdminDepartment> {
    return this.http.post<AdminDepartment>(`${this.baseUrl}/departments`, input);
  }

  updateDepartment(departmentId: string, input: SaveAdminDepartment): Observable<AdminDepartment> {
    return this.http.patch<AdminDepartment>(
      `${this.baseUrl}/departments/${encodeURIComponent(departmentId)}`,
      input,
    );
  }
}
