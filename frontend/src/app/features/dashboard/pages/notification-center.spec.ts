import { TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of } from 'rxjs';
import { NotificationService } from '../../../core/notifications/notification.service';
import { NotificationCenter } from './notification-center';

describe('NotificationCenter', () => {
  it('renders unread notifications with a resource action', async () => {
    await TestBed.configureTestingModule({
      imports: [NotificationCenter],
      providers: [
        provideRouter([]),
        {
          provide: NotificationService,
          useValue: {
            list: () =>
              of({
                items: [
                  {
                    id: 'notification-id',
                    eventType: 'SUBMITTED',
                    resourceType: 'purchase_request',
                    resourceId: 'request-id',
                    title: 'Phiếu mua sắm cần phê duyệt',
                    body: 'PR-2026-000001 - Laptop',
                    createdAt: '2026-08-15T00:00:00Z',
                    readAt: null,
                  },
                ],
                page: 1,
                pageSize: 20,
                total: 1,
                pages: 1,
                unreadCount: 1,
              }),
            markRead: () => of({ read: true }),
            markAllRead: () => of({ markedRead: 1 }),
          },
        },
      ],
    }).compileComponents();
    const fixture = TestBed.createComponent(NotificationCenter);
    fixture.detectChanges();
    const text = (fixture.nativeElement as HTMLElement).textContent ?? '';
    expect(text).toContain('Phiếu mua sắm cần phê duyệt');
    expect(text).toContain('Chưa đọc');
    expect(text).toContain('Mở nội dung');
  });
});
