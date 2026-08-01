import {
  ChangeDetectionStrategy,
  Component,
  DestroyRef,
  ElementRef,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import {
  FormArray,
  FormControl,
  FormGroup,
  NonNullableFormBuilder,
  ReactiveFormsModule,
  Validators,
} from '@angular/forms';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { firstValueFrom } from 'rxjs';
import { problemMessage, problemViolations } from '../../data-access/problem-details';
import {
  CreatePurchaseRequest,
  PurchaseRequestItem,
  UpdatePurchaseRequest,
} from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';

type ItemForm = FormGroup<{
  description: FormControl<string>;
  quantity: FormControl<string>;
  unit: FormControl<string>;
  unitPrice: FormControl<string>;
}>;

const quantityPattern = /^(?!(?:0|0\.0{1,4})$)(0|[1-9][0-9]{0,10})(\.[0-9]{1,4})?$/;
const unitPricePattern = /^(0|[1-9][0-9]{0,14})(\.[0-9]{1,4})?$/;

@Component({
  selector: 'app-purchase-request-create',
  imports: [ReactiveFormsModule, RouterLink, HlmButton, ...HlmCardImports, MoneyPipe],
  templateUrl: './purchase-request-create.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PurchaseRequestCreate {
  private readonly formBuilder = inject(NonNullableFormBuilder);
  private readonly procurement = inject(ProcurementService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly destroyRef = inject(DestroyRef);
  private readonly formElement = viewChild<ElementRef<HTMLFormElement>>('requestForm');
  private readonly submitErrorElement = viewChild<ElementRef<HTMLDivElement>>('submitErrorBanner');

  readonly form = this.formBuilder.group({
    title: ['', [Validators.required, Validators.minLength(3), Validators.maxLength(255)]],
    reason: ['', [Validators.required, Validators.minLength(10), Validators.maxLength(5000)]],
    currency: ['VND', [Validators.required, Validators.pattern(/^[A-Za-z]{3}$/)]],
    costCenter: [
      'CC-GENERAL',
      [Validators.required, Validators.minLength(1), Validators.maxLength(100)],
    ],
    items: this.formBuilder.array<ItemForm>([this.createItemForm()]),
  });

  readonly items = this.form.controls.items;
  readonly requestId = this.route.snapshot.paramMap.get('requestId');
  readonly editing = this.requestId !== null;
  readonly loading = signal(this.editing);
  readonly expectedVersion = signal<number | null>(null);
  readonly submitting = signal(false);
  readonly submitError = signal<string | null>(null);
  readonly serverErrors = signal<Record<string, string>>({});
  readonly estimatedTotal = signal('0');

  constructor() {
    this.form.valueChanges
      .pipe(takeUntilDestroyed())
      .subscribe(() => this.calculateEstimatedTotal());
    this.calculateEstimatedTotal();
    if (this.requestId) {
      void this.loadExisting(this.requestId);
    }
  }

  addItem(): void {
    if (this.items.length >= 100) {
      return;
    }
    this.items.push(this.createItemForm());
  }

  removeItem(index: number): void {
    if (this.items.length <= 1) {
      return;
    }
    this.items.removeAt(index);
  }

  async submit(): Promise<void> {
    this.submitError.set(null);
    this.serverErrors.set({});
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      this.submitError.set('Vui lòng kiểm tra các trường bắt buộc và định dạng số tiền.');
      queueMicrotask(() => {
        this.formElement()
          ?.nativeElement.querySelector<HTMLElement>('[aria-invalid="true"]')
          ?.focus();
      });
      return;
    }

    this.submitting.set(true);
    try {
      const value = this.form.getRawValue();
      const input: CreatePurchaseRequest = {
        title: value.title.trim(),
        reason: value.reason.trim(),
        currency: value.currency.trim().toUpperCase(),
        costCenter: value.costCenter.trim(),
        items: value.items.map((item) => ({
          description: item.description.trim(),
          quantity: item.quantity.trim(),
          unit: item.unit.trim(),
          unitPrice: item.unitPrice.trim(),
        })),
      };
      const result =
        this.requestId && this.expectedVersion()
          ? await firstValueFrom(
              this.procurement.update(this.requestId, {
                ...input,
                expectedVersion: this.expectedVersion()!,
              } satisfies UpdatePurchaseRequest),
            )
          : await firstValueFrom(this.procurement.create(input));
      await this.router.navigate(['/purchase-requests', result.id], {
        queryParams: { [this.editing ? 'updated' : 'created']: 1 },
      });
    } catch (error: unknown) {
      this.serverErrors.set(problemViolations(error));
      this.submitError.set(
        problemMessage(
          error,
          this.editing
            ? 'Không cập nhật được phiếu. Hãy tải lại và thử lại.'
            : 'Không tạo được phiếu. Hãy thử lại.',
        ),
      );
      queueMicrotask(() => this.submitErrorElement()?.nativeElement.focus());
    } finally {
      this.submitting.set(false);
    }
  }

  private createItemForm(item?: PurchaseRequestItem): ItemForm {
    return this.formBuilder.group({
      description: [
        item?.description ?? '',
        [Validators.required, Validators.minLength(2), Validators.maxLength(500)],
      ],
      quantity: [item?.quantity ?? '1', [Validators.required, Validators.pattern(quantityPattern)]],
      unit: [item?.unit ?? 'chiếc', [Validators.required, Validators.maxLength(50)]],
      unitPrice: [
        item?.unitPrice ?? '0',
        [Validators.required, Validators.pattern(unitPricePattern)],
      ],
    });
  }

  private async loadExisting(requestId: string): Promise<void> {
    this.form.disable();
    this.submitError.set(null);
    try {
      const request = await firstValueFrom(this.procurement.get(requestId));
      if (request.status !== 'DRAFT' && request.status !== 'CHANGES_REQUESTED') {
        this.submitError.set('Chỉ có thể sửa phiếu ở trạng thái Bản nháp hoặc Yêu cầu chỉnh sửa.');
        return;
      }
      this.expectedVersion.set(request.version);
      this.form.patchValue({
        title: request.title,
        reason: request.reason,
        currency: request.currency,
        costCenter: request.costCenter,
      });
      this.items.clear();
      for (const item of request.items ?? []) {
        this.items.push(this.createItemForm(item));
      }
      if (this.items.length === 0) {
        this.items.push(this.createItemForm());
      }
      this.form.enable();
      this.calculateEstimatedTotal();
    } catch (error: unknown) {
      this.submitError.set(
        problemMessage(
          error,
          'Không tải được phiếu để chỉnh sửa. Phiếu có thể nằm ngoài phạm vi của bạn.',
        ),
      );
    } finally {
      this.loading.set(false);
    }
  }

  private calculateEstimatedTotal(): void {
    const total = this.items.controls.reduce((sum, item) => {
      const quantity = Number(item.controls.quantity.value);
      const unitPrice = Number(item.controls.unitPrice.value);
      if (!Number.isFinite(quantity) || !Number.isFinite(unitPrice)) {
        return sum;
      }
      return sum + quantity * unitPrice;
    }, 0);
    this.estimatedTotal.set(Number.isFinite(total) ? total.toFixed(4) : '0');
  }
}
