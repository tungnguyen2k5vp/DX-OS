import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { APP_CONFIG } from '../config/app-config';
import { NotificationService } from './notification.service';

describe('NotificationService', () => {
  let service: NotificationService;
  let http: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: APP_CONFIG, useValue: { apiBaseUrl: 'http://api.test' } },
      ],
    });
    service = TestBed.inject(NotificationService);
    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => http.verify());

  it('loads notifications and updates the global unread count', () => {
    service.list(2, 10, true).subscribe();
    const request = http.expectOne(
      (candidate) =>
        candidate.url === 'http://api.test/api/v1/me/notifications' &&
        candidate.params.get('page') === '2' &&
        candidate.params.get('pageSize') === '10' &&
        candidate.params.get('unreadOnly') === 'true',
    );
    request.flush({ items: [], page: 2, pageSize: 10, total: 0, pages: 0, unreadCount: 4 });
    expect(service.unreadCount()).toBe(4);
  });

  it('marks one or all notifications read', () => {
    service.list().subscribe();
    http
      .expectOne((request) => request.url.endsWith('/me/notifications'))
      .flush({
        items: [],
        page: 1,
        pageSize: 20,
        total: 0,
        pages: 0,
        unreadCount: 2,
      });

    service.markRead('notification-id').subscribe();
    const single = http.expectOne('http://api.test/api/v1/me/notifications/notification-id/read');
    expect(single.request.method).toBe('POST');
    single.flush({ read: true });
    expect(service.unreadCount()).toBe(1);

    service.markAllRead().subscribe();
    const all = http.expectOne('http://api.test/api/v1/me/notifications/read-all');
    expect(all.request.method).toBe('POST');
    all.flush({ markedRead: 1 });
    expect(service.unreadCount()).toBe(0);
  });
});
