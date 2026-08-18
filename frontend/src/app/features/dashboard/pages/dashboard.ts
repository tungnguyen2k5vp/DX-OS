import { DatePipe } from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  DestroyRef,
  inject,
  signal,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { catchError, forkJoin, Observable, of } from 'rxjs';
import { AuthService } from '../../../core/auth/auth.service';
import { CurrentUser, IdentityService } from '../../../core/http/identity.service';
import {
  BudgetDashboard,
  PurchaseRequest,
  PurchaseRequestPage,
  PurchaseRequestStatus,
} from '../../procurement/data-access/procurement.models';
import { ProcurementService } from '../../procurement/data-access/procurement.service';
import { MoneyPipe } from '../../procurement/ui/money.pipe';
import { PurchaseRequestStatusBadge } from '../../procurement/ui/purchase-request-status-badge';
import { ProcurementReportDashboard } from '../../reporting/data-access/reporting.models';
import { ReportingService } from '../../reporting/data-access/reporting.service';

interface RoleWorkspace {
  name: string;
  mission: string;
  responsibilities: string[];
}

const roleWorkspaces: Record<string, RoleWorkspace> = {
  employee: {
    name: 'Nhân viên đề xuất',
    mission: 'Tạo yêu cầu mua sắm đầy đủ, theo dõi tiến độ và bổ sung khi được yêu cầu.',
    responsibilities: ['Tạo và gửi phiếu', 'Bổ sung hồ sơ bị trả lại', 'Theo dõi lịch sử xử lý'],
  },
  department_manager: {
    name: 'Trưởng bộ phận',
    mission: 'Kiểm tra nhu cầu của phòng ban trước khi chuyển hồ sơ sang tài chính.',
    responsibilities: ['Duyệt nhu cầu phòng ban', 'Yêu cầu chỉnh sửa', 'Theo dõi phiếu của đơn vị'],
  },
  finance: {
    name: 'Tài chính',
    mission: 'Kiểm soát ngân sách, hồ sơ và ra quyết định tài chính cuối cùng.',
    responsibilities: ['Duyệt tài chính', 'Điều chỉnh hạn mức', 'Theo dõi KPI và SLA'],
  },
  auditor: {
    name: 'Kiểm toán',
    mission: 'Đọc dữ liệu độc lập để kiểm tra ngân sách, quy trình và dấu vết thay đổi.',
    responsibilities: ['Xem ngân sách', 'Đối chiếu báo cáo', 'Kiểm tra bằng chứng và lịch sử'],
  },
  dx_admin: {
    name: 'Quản trị DX-OS',
    mission: 'Giám sát số liệu vận hành và hỗ trợ phân quyền trên hệ thống danh tính.',
    responsibilities: [
      'Xem báo cáo toàn tổ chức',
      'Giám sát SLA',
      'Quản trị tài khoản trên Keycloak',
    ],
  },
  ai_operator: {
    name: 'Vận hành AI',
    mission: 'Theo dõi nền tảng và chuẩn bị dữ liệu cho mô-đun AI ở giai đoạn tiếp theo.',
    responsibilities: ['Kiểm tra kết nối', 'Đánh giá chất lượng dữ liệu', 'Theo dõi kế hoạch AI'],
  },
};

const defaultWorkspace: RoleWorkspace = {
  name: 'Người dùng DX-OS',
  mission: 'Tài khoản đã đăng nhập nhưng chưa có vai trò nghiệp vụ phù hợp.',
  responsibilities: ['Liên hệ quản trị viên để được gán vai trò'],
};

@Component({
  selector: 'app-dashboard',
  imports: [
    DatePipe,
    RouterLink,
    HlmBadge,
    HlmButton,
    ...HlmCardImports,
    MoneyPipe,
    PurchaseRequestStatusBadge,
  ],
  templateUrl: './dashboard.html',
  styleUrl: './dashboard.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Dashboard {
  private readonly identity = inject(IdentityService);
  private readonly procurement = inject(ProcurementService);
  private readonly reporting = inject(ReportingService);
  private readonly auth = inject(AuthService);
  private readonly destroyRef = inject(DestroyRef);

  readonly currentUser = signal<CurrentUser | null>(null);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly metricsLoading = signal(true);
  readonly metricsPartialFailure = signal(false);
  readonly requestTotal = signal<number | null>(null);
  readonly actionRequired = signal<number | null>(null);
  readonly approvedTotal = signal<number | null>(null);
  readonly budgetAlertCount = signal<number | null>(null);
  readonly reportDashboard = signal<ProcurementReportDashboard | null>(null);
  readonly recentRequests = signal<PurchaseRequest[]>([]);

  readonly roles = computed(() => this.auth.roles());
  readonly canBrowseRequests = computed(() =>
    this.hasAnyRole('employee', 'department_manager', 'finance', 'auditor'),
  );
  readonly canCreate = computed(() => this.hasAnyRole('employee', 'department_manager'));
  readonly canReview = computed(() => this.hasAnyRole('department_manager', 'finance'));
  readonly canAccessBudgets = computed(() => this.hasAnyRole('finance', 'auditor'));
  readonly canAccessReports = computed(() => this.hasAnyRole('finance', 'auditor', 'dx_admin'));
  readonly workspace = computed(
    () =>
      this.roles()
        .map((role) => roleWorkspaces[role])
        .find(Boolean) ?? defaultWorkspace,
  );
  readonly actionLabel = computed(() => {
    if (this.roles().includes('finance')) {
      return 'Chờ duyệt tài chính';
    }
    if (this.roles().includes('department_manager')) {
      return 'Chờ trưởng bộ phận';
    }
    if (this.roles().includes('employee')) {
      return 'Cần tôi bổ sung';
    }
    return 'Cần tôi xử lý';
  });
  readonly completedValue = computed(
    () => this.reportDashboard()?.summary.approvedCount ?? this.approvedTotal(),
  );
  readonly attentionLabel = computed(() =>
    this.canAccessBudgets() ? 'Cảnh báo ngân sách' : 'Phiếu quá SLA (30 ngày)',
  );
  readonly attentionValue = computed(() =>
    this.canAccessBudgets()
      ? this.budgetAlertCount()
      : (this.reportDashboard()?.summary.slaBreachedCount ?? null),
  );

  constructor() {
    this.identity
      .getCurrentUser()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (user) => {
          this.currentUser.set(user);
          this.loading.set(false);
        },
        error: () => {
          this.error.set('Không gọi được Go API. Hãy kiểm tra container api và đăng nhập lại.');
          this.loading.set(false);
        },
      });

    this.loadWorkspaceMetrics();
  }

  private loadWorkspaceMetrics(): void {
    const actionStatus = this.actionStatus();
    const requests$ = this.canBrowseRequests()
      ? this.recover(this.procurement.list({ page: 1, pageSize: 5 }))
      : of(null);
    const action$ = actionStatus
      ? this.recover(this.procurement.list({ page: 1, pageSize: 1, status: actionStatus }))
      : of(null);
    const approved$ = this.canBrowseRequests()
      ? this.recover(this.procurement.list({ page: 1, pageSize: 1, status: 'APPROVED' }))
      : of(null);
    const budget$ = this.canAccessBudgets()
      ? this.recover(this.procurement.budgetDashboard())
      : of(null);
    const reports$ = this.canAccessReports()
      ? this.recover(
          this.reporting.procurementDashboard({ from: dateInput(-29), to: dateInput(0) }),
        )
      : of(null);

    forkJoin({
      requests: requests$,
      action: action$,
      approved: approved$,
      budget: budget$,
      reports: reports$,
    })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe(({ requests, action, approved, budget, reports }) => {
        this.applyRequestMetrics(requests, action, approved);
        this.budgetAlertCount.set(budget?.alertCount ?? null);
        this.reportDashboard.set(reports);
        this.metricsLoading.set(false);
      });
  }

  private applyRequestMetrics(
    requests: PurchaseRequestPage | null,
    action: PurchaseRequestPage | null,
    approved: PurchaseRequestPage | null,
  ): void {
    this.requestTotal.set(requests?.total ?? null);
    this.recentRequests.set(requests?.items ?? []);
    this.actionRequired.set(action?.total ?? null);
    this.approvedTotal.set(approved?.total ?? null);
  }

  private actionStatus(): PurchaseRequestStatus | null {
    if (this.roles().includes('finance')) {
      return 'MANAGER_APPROVED';
    }
    if (this.roles().includes('department_manager')) {
      return 'SUBMITTED';
    }
    if (this.roles().includes('employee')) {
      return 'CHANGES_REQUESTED';
    }
    return null;
  }

  private recover<T>(source: Observable<T>): Observable<T | null> {
    return source.pipe(
      catchError(() => {
        this.metricsPartialFailure.set(true);
        return of(null);
      }),
    );
  }

  private hasAnyRole(...roles: string[]): boolean {
    return roles.some((role) => this.roles().includes(role));
  }
}

function dateInput(offsetDays: number): string {
  const date = new Date();
  date.setDate(date.getDate() + offsetDays);
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}
