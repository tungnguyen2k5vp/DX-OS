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
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { AuthService } from '../../../../core/auth/auth.service';
import { problemMessage } from '../../data-access/problem-details';
import {
  isPurchaseRequestStatus,
  PurchaseRequest,
  PurchaseRequestStatus,
  purchaseRequestStatuses,
} from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';
import {
  purchaseRequestStatusLabels,
  PurchaseRequestStatusBadge,
} from '../../ui/purchase-request-status-badge';

@Component({
  selector: 'app-purchase-request-list',
  imports: [
    DatePipe,
    RouterLink,
    HlmButton,
    ...HlmCardImports,
    MoneyPipe,
    PurchaseRequestStatusBadge,
  ],
  templateUrl: './purchase-request-list.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PurchaseRequestList {
  private readonly procurement = inject(ProcurementService);
  private readonly auth = inject(AuthService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);
  private requestGeneration = 0;

  readonly statusOptions = purchaseRequestStatuses.map((value) => ({
    value,
    label: purchaseRequestStatusLabels[value],
  }));
  readonly items = signal<PurchaseRequest[]>([]);
  readonly page = signal(1);
  readonly pageSize = signal(20);
  readonly total = signal(0);
  readonly pages = signal(0);
  readonly status = signal<PurchaseRequestStatus | ''>('');
  readonly search = signal('');
  readonly department = signal('');
  readonly costCenter = signal('');
  readonly requester = signal('');
  readonly from = signal('');
  readonly to = signal('');
  readonly minAmount = signal('');
  readonly maxAmount = signal('');
  readonly sort = signal<'createdAt' | 'updatedAt' | 'amount' | 'code'>('createdAt');
  readonly direction = signal<'asc' | 'desc'>('desc');
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly canCreate = computed(() => {
    const roles = this.auth.roles();
    return roles.includes('employee') || roles.includes('department_manager');
  });

  constructor() {
    this.route.queryParamMap.pipe(takeUntilDestroyed()).subscribe((params) => {
      this.page.set(parsePositiveInteger(params.get('page'), 1));
      this.pageSize.set(parsePageSize(params.get('pageSize')));
      const rawStatus = params.get('status')?.toUpperCase() ?? '';
      this.status.set(isPurchaseRequestStatus(rawStatus) ? rawStatus : '');
      this.search.set(params.get('search') ?? '');
      this.department.set(params.get('department') ?? '');
      this.costCenter.set(params.get('costCenter') ?? '');
      this.requester.set(params.get('requester') ?? '');
      this.from.set(params.get('from') ?? '');
      this.to.set(params.get('to') ?? '');
      this.minAmount.set(params.get('minAmount') ?? '');
      this.maxAmount.set(params.get('maxAmount') ?? '');
      this.sort.set(parseSort(params.get('sort')));
      this.direction.set(params.get('direction') === 'asc' ? 'asc' : 'desc');
      this.items.set([]);
      this.load();
    });
  }

  retry(): void {
    this.load();
  }

  changeStatus(event: Event): void {
    const value = (event.target as HTMLSelectElement).value;
    this.status.set(isPurchaseRequestStatus(value) ? value : '');
    this.applyFilters();
  }

  applyFilters(): void {
    void this.router.navigate([], {
      relativeTo: this.route,
      queryParams: this.filterQuery(1),
    });
  }

  clearFilters(): void {
    this.status.set('');
    this.search.set('');
    this.department.set('');
    this.costCenter.set('');
    this.requester.set('');
    this.from.set('');
    this.to.set('');
    this.minAmount.set('');
    this.maxAmount.set('');
    this.sort.set('createdAt');
    this.direction.set('desc');
    this.applyFilters();
  }

  goToPage(targetPage: number): void {
    if (targetPage < 1 || targetPage > this.pages() || targetPage === this.page()) {
      return;
    }
    void this.router.navigate([], {
      relativeTo: this.route,
      queryParams: this.filterQuery(targetPage),
    });
  }

  private load(): void {
    const generation = ++this.requestGeneration;
    this.loading.set(true);
    this.error.set(null);

    this.procurement
      .list({
        page: this.page(),
        pageSize: this.pageSize(),
        status: this.status() || undefined,
        search: this.search() || undefined,
        department: this.department() || undefined,
        costCenter: this.costCenter() || undefined,
        requester: this.requester() || undefined,
        from: this.from() || undefined,
        to: this.to() || undefined,
        minAmount: this.minAmount() || undefined,
        maxAmount: this.maxAmount() || undefined,
        sort: this.sort(),
        direction: this.direction(),
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (generation !== this.requestGeneration) {
            return;
          }
          this.items.set(result.items);
          this.total.set(result.total);
          this.pages.set(result.pages);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.requestGeneration) {
            return;
          }
          this.error.set(
            problemMessage(error, 'Không tải được danh sách phiếu mua sắm. Hãy thử lại.'),
          );
          this.loading.set(false);
        },
      });
  }

  private filterQuery(page: number): Record<string, string | number | null> {
    return {
      page,
      pageSize: this.pageSize(),
      status: this.status() || null,
      search: this.search().trim() || null,
      department: this.department().trim() || null,
      costCenter: this.costCenter().trim() || null,
      requester: this.requester().trim() || null,
      from: this.from() || null,
      to: this.to() || null,
      minAmount: this.minAmount().trim() || null,
      maxAmount: this.maxAmount().trim() || null,
      sort: this.sort(),
      direction: this.direction(),
    };
  }
}

function parsePositiveInteger(value: string | null, fallback: number): number {
  const parsed = Number.parseInt(value ?? '', 10);
  return Number.isInteger(parsed) && parsed >= 1 ? parsed : fallback;
}

function parsePageSize(value: string | null): number {
  const parsed = parsePositiveInteger(value, 20);
  return Math.min(parsed, 100);
}

function parseSort(value: string | null): 'createdAt' | 'updatedAt' | 'amount' | 'code' {
  return value === 'updatedAt' || value === 'amount' || value === 'code' ? value : 'createdAt';
}
