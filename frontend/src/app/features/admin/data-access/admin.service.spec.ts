import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { APP_CONFIG } from '../../../core/config/app-config';
import { AdminService } from './admin.service';

describe('AdminService', () => {
  it('loads the center and updates a user with optimistic locking', () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: APP_CONFIG, useValue: { apiBaseUrl: 'http://api.test' } },
      ],
    });
    const service = TestBed.inject(AdminService);
    const http = TestBed.inject(HttpTestingController);
    service.center().subscribe();
    http.expectOne('http://api.test/api/v1/admin/center').flush({ users: [], departments: [] });

    const input = {
      displayName: 'Demo User',
      email: 'demo@example.com',
      departmentId: 'department-id',
      active: true,
      expectedVersion: 2,
    };
    service.updateUser('user-id', input).subscribe();
    const request = http.expectOne('http://api.test/api/v1/admin/users/user-id');
    expect(request.request.method).toBe('PATCH');
    expect(request.request.body).toEqual(input);
    request.flush({ id: 'user-id', version: 3 });
    http.verify();
  });
});
