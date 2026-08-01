import { inject, Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { APP_CONFIG } from '../config/app-config';

export interface CurrentUser {
  subject: string;
  username: string;
  email?: string;
  roles: string[];
}

@Injectable({ providedIn: 'root' })
export class IdentityService {
  private readonly http = inject(HttpClient);
  private readonly config = inject(APP_CONFIG);

  getCurrentUser(): Observable<CurrentUser> {
    return this.http.get<CurrentUser>(`${this.config.apiBaseUrl}/api/v1/me`);
  }
}
