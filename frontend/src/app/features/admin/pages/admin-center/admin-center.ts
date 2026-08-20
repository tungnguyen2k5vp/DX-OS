import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { problemMessage } from '../../../procurement/data-access/problem-details';
import { AdminCenterModel, AdminDepartment, AdminUser } from '../../data-access/admin.models';
import { AdminService } from '../../data-access/admin.service';
import { revealWorkspace } from '../../../../shared/utils/reveal-workspace';

@Component({
  selector: 'app-admin-center',
  imports: [HlmBadge, HlmButton, ...HlmCardImports],
  templateUrl: './admin-center.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminCenterPage {
  private readonly service = inject(AdminService);
  private readonly destroyRef = inject(DestroyRef);

  readonly center = signal<AdminCenterModel | null>(null);
  readonly loading = signal(true);
  readonly busy = signal(false);
  readonly error = signal<string | null>(null);
  readonly editingUser = signal<AdminUser | null>(null);
  readonly editDisplayName = signal('');
  readonly editEmail = signal('');
  readonly editDepartmentId = signal('');
  readonly editActive = signal(true);
  readonly departmentCode = signal('');
  readonly departmentName = signal('');
  readonly departmentCostCenter = signal('');
  readonly departmentParentId = signal('');

  constructor() {
    this.load();
  }

  editUser(item: AdminUser): void {
    this.editingUser.set(item);
    this.editDisplayName.set(item.displayName);
    this.editEmail.set(item.email);
    this.editDepartmentId.set(item.departmentId);
    this.editActive.set(item.active);
    revealWorkspace('user-editor-workspace');
  }

  saveUser(): void {
    const item = this.editingUser();
    if (!item) return;
    this.busy.set(true);
    this.service
      .updateUser(item.id, {
        displayName: this.editDisplayName(),
        email: this.editEmail(),
        departmentId: this.editDepartmentId(),
        active: this.editActive(),
        expectedVersion: item.version,
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.editingUser.set(null);
          this.busy.set(false);
          this.load();
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không lưu được người dùng.'));
          this.busy.set(false);
        },
      });
  }

  createDepartment(): void {
    this.busy.set(true);
    this.service
      .createDepartment({
        code: this.departmentCode(),
        name: this.departmentName(),
        costCenter: this.departmentCostCenter(),
        parentId: this.departmentParentId(),
        active: true,
        expectedVersion: 0,
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.departmentCode.set('');
          this.departmentName.set('');
          this.departmentCostCenter.set('');
          this.departmentParentId.set('');
          this.busy.set(false);
          this.load();
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không tạo được phòng ban.'));
          this.busy.set(false);
        },
      });
  }

  toggleDepartment(item: AdminDepartment): void {
    this.busy.set(true);
    this.service
      .updateDepartment(item.id, {
        code: item.code,
        name: item.name,
        costCenter: item.costCenter,
        parentId: item.parentId || '',
        active: !item.active,
        expectedVersion: item.version,
      })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.busy.set(false);
          this.load();
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không đổi được trạng thái phòng ban.'));
          this.busy.set(false);
        },
      });
  }

  private load(): void {
    this.loading.set(true);
    this.error.set(null);
    this.service
      .center()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          this.center.set(result);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không tải được trung tâm quản trị.'));
          this.loading.set(false);
        },
      });
  }
}
