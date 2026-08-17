import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { firstValueFrom } from 'rxjs';
import { problemMessage } from '../../data-access/problem-details';
import {
  Supplier,
  SupplierInput,
  SupplierComplianceStatus,
  SupplierRiskLevel,
  SupplierStatus,
} from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';

@Component({
  selector: 'app-supplier-directory',
  imports: [HlmBadge, HlmButton, ...HlmCardImports],
  templateUrl: './supplier-directory.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SupplierDirectory {
  private readonly procurement = inject(ProcurementService);
  private readonly destroyRef = inject(DestroyRef);
  private generation = 0;

  readonly items = signal<Supplier[]>([]);
  readonly canManage = signal(false);
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly saving = signal(false);
  readonly editing = signal<Supplier | null>(null);
  readonly formOpen = signal(false);
  readonly code = signal('');
  readonly name = signal('');
  readonly taxCode = signal('');
  readonly contactName = signal('');
  readonly email = signal('');
  readonly phone = signal('');
  readonly status = signal<SupplierStatus>('ACTIVE');
  readonly riskLevel = signal<SupplierRiskLevel>('LOW');
  readonly address = signal('');
  readonly bankName = signal('');
  readonly bankAccountNumber = signal('');
  readonly contractReference = signal('');
  readonly contractExpiresOn = signal('');
  readonly complianceStatus = signal<SupplierComplianceStatus>('PENDING');
  readonly performanceScore = signal('');
  readonly businessNote = signal('');

  constructor() {
    this.load();
  }

  openCreate(): void {
    this.resetForm();
    this.formOpen.set(true);
  }

  openEdit(supplier: Supplier): void {
    this.editing.set(supplier);
    this.code.set(supplier.code);
    this.name.set(supplier.name);
    this.taxCode.set(supplier.taxCode ?? '');
    this.contactName.set(supplier.contactName ?? '');
    this.email.set(supplier.email ?? '');
    this.phone.set(supplier.phone ?? '');
    this.status.set(supplier.status);
    this.riskLevel.set(supplier.riskLevel);
    this.address.set(supplier.address ?? '');
    this.bankName.set(supplier.bankName ?? '');
    this.bankAccountNumber.set(supplier.bankAccountNumber ?? '');
    this.contractReference.set(supplier.contractReference ?? '');
    this.contractExpiresOn.set(supplier.contractExpiresOn ?? '');
    this.complianceStatus.set(supplier.complianceStatus);
    this.performanceScore.set(supplier.performanceScore ?? '');
    this.businessNote.set(supplier.businessNote ?? '');
    this.error.set(null);
    this.formOpen.set(true);
  }

  cancelForm(): void {
    this.formOpen.set(false);
    this.resetForm();
  }

  async save(): Promise<void> {
    if (this.code().trim().length < 2 || this.name().trim().length < 2) {
      this.error.set('Mã và tên nhà cung cấp phải có ít nhất 2 ký tự.');
      return;
    }
    const current = this.editing();
    const input: SupplierInput = {
      code: this.code(),
      name: this.name(),
      taxCode: this.taxCode(),
      contactName: this.contactName(),
      email: this.email(),
      phone: this.phone(),
      status: this.status(),
      riskLevel: this.riskLevel(),
      address: this.address(),
      bankName: this.bankName(),
      bankAccountNumber: this.bankAccountNumber(),
      contractReference: this.contractReference(),
      contractExpiresOn: this.contractExpiresOn(),
      complianceStatus: this.complianceStatus(),
      performanceScore: this.performanceScore(),
      businessNote: this.businessNote(),
      expectedVersion: current?.version,
    };
    this.saving.set(true);
    this.error.set(null);
    try {
      if (current) {
        await firstValueFrom(this.procurement.updateSupplier(current.id, input));
      } else {
        await firstValueFrom(this.procurement.createSupplier(input));
      }
      this.cancelForm();
      this.load();
    } catch (error: unknown) {
      this.error.set(problemMessage(error, 'Không lưu được nhà cung cấp.'));
    } finally {
      this.saving.set(false);
    }
  }

  riskVariant(risk: SupplierRiskLevel): 'destructive' | 'secondary' | 'outline' {
    if (risk === 'HIGH') return 'destructive';
    return risk === 'MEDIUM' ? 'secondary' : 'outline';
  }

  private load(): void {
    const generation = ++this.generation;
    this.loading.set(true);
    this.procurement
      .suppliers()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (generation !== this.generation) return;
          this.items.set(result.items);
          this.canManage.set(result.canManage);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.generation) return;
          this.error.set(problemMessage(error, 'Không tải được danh mục nhà cung cấp.'));
          this.loading.set(false);
        },
      });
  }

  private resetForm(): void {
    this.editing.set(null);
    this.code.set('');
    this.name.set('');
    this.taxCode.set('');
    this.contactName.set('');
    this.email.set('');
    this.phone.set('');
    this.status.set('ACTIVE');
    this.riskLevel.set('LOW');
    this.address.set('');
    this.bankName.set('');
    this.bankAccountNumber.set('');
    this.contractReference.set('');
    this.contractExpiresOn.set('');
    this.complianceStatus.set('PENDING');
    this.performanceScore.set('');
    this.businessNote.set('');
    this.error.set(null);
  }
}
