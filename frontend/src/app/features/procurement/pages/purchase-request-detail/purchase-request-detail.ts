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
import { ActivatedRoute, RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { firstValueFrom } from 'rxjs';
import { AuthService } from '../../../../core/auth/auth.service';
import { problemMessage } from '../../data-access/problem-details';
import {
  BudgetCheck,
  AttachmentDocumentType,
  PurchaseRequest,
  PurchaseRequestAttachmentList,
  PurchaseRequestAction,
  PurchaseRequestComment,
  PurchaseRequestTimelineEvent,
} from '../../data-access/procurement.models';
import { ProcurementService } from '../../data-access/procurement.service';
import { MoneyPipe } from '../../ui/money.pipe';
import { PurchaseRequestStatusBadge } from '../../ui/purchase-request-status-badge';

const timelineLabels: Record<string, string> = {
  DRAFT_CREATED: 'Đã tạo bản nháp',
  DRAFT_UPDATED: 'Đã cập nhật bản nháp',
  SUBMITTED: 'Đã gửi trưởng bộ phận',
  RESUBMITTED: 'Đã gửi duyệt lại',
  MANAGER_APPROVED: 'Trưởng bộ phận đã phê duyệt',
  FINANCE_APPROVED: 'Tài chính đã phê duyệt',
  BUDGET_RESERVED: 'Đã giữ ngân sách',
  BUDGET_COMMITTED: 'Đã cam kết ngân sách',
  BUDGET_RELEASED: 'Đã hoàn trả ngân sách',
  CHANGES_REQUESTED: 'Đã yêu cầu chỉnh sửa',
  REJECTED: 'Đã từ chối phiếu',
  CANCELLED: 'Đã hủy phiếu',
};

const timelineRoleLabels: Record<string, string> = {
  employee: 'Nhân viên',
  department_manager: 'Trưởng bộ phận',
  finance: 'Tài chính',
  auditor: 'Kiểm toán',
  dx_admin: 'Quản trị',
};

interface TimelineViewEvent extends PurchaseRequestTimelineEvent {
  label: string;
  actorRoleLabel: string;
}

@Component({
  selector: 'app-purchase-request-detail',
  imports: [
    DatePipe,
    RouterLink,
    HlmBadge,
    HlmButton,
    ...HlmCardImports,
    MoneyPipe,
    PurchaseRequestStatusBadge,
  ],
  templateUrl: './purchase-request-detail.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PurchaseRequestDetail {
  private readonly procurement = inject(ProcurementService);
  private readonly auth = inject(AuthService);
  private readonly route = inject(ActivatedRoute);
  private readonly destroyRef = inject(DestroyRef);
  private requestGeneration = 0;
  private timelineGeneration = 0;
  private budgetGeneration = 0;
  private attachmentGeneration = 0;
  private commentGeneration = 0;

  readonly request = signal<PurchaseRequest | null>(null);
  readonly requestId = signal('');
  readonly loading = signal(true);
  readonly error = signal<string | null>(null);
  readonly created = signal(this.route.snapshot.queryParamMap.get('created') === '1');
  readonly updated = signal(this.route.snapshot.queryParamMap.get('updated') === '1');
  readonly selectedAction = signal<PurchaseRequestAction | null>(null);
  readonly transitionComment = signal('');
  readonly transitioning = signal(false);
  readonly transitionError = signal<string | null>(null);
  readonly transitionSuccess = signal<string | null>(null);
  readonly timeline = signal<PurchaseRequestTimelineEvent[]>([]);
  readonly timelinePage = signal(1);
  readonly timelinePageSize = 20;
  readonly timelineTotal = signal(0);
  readonly timelinePages = signal(0);
  readonly timelineLoading = signal(true);
  readonly timelineError = signal<string | null>(null);
  readonly budget = signal<BudgetCheck | null>(null);
  readonly budgetLoading = signal(true);
  readonly budgetError = signal<string | null>(null);
  readonly attachments = signal<PurchaseRequestAttachmentList | null>(null);
  readonly attachmentsLoading = signal(true);
  readonly attachmentError = signal<string | null>(null);
  readonly selectedDocumentType = signal<AttachmentDocumentType>('QUOTATION');
  readonly selectedFile = signal<File | null>(null);
  readonly uploadingAttachment = signal(false);
  readonly deletingAttachmentId = signal<string | null>(null);
  readonly comments = signal<PurchaseRequestComment[]>([]);
  readonly commentsLoading = signal(true);
  readonly commentsError = signal<string | null>(null);
  readonly commentBody = signal('');
  readonly submittingComment = signal(false);
  readonly canComment = computed(() => {
    const roles = this.auth.roles();
    return (
      !roles.includes('auditor') &&
      (roles.includes('employee') ||
        roles.includes('department_manager') ||
        roles.includes('finance'))
    );
  });
  readonly canEditAttachments = computed(() => {
    const request = this.request();
    return (
      this.isOwner() && (request?.status === 'DRAFT' || request?.status === 'CHANGES_REQUESTED')
    );
  });
  readonly budgetResultLabel = computed(() => {
    const result = this.budget()?.result ?? 'NOT_CONFIGURED';
    return {
      NOT_CONFIGURED: 'Chưa cấu hình',
      AVAILABLE: 'Đủ ngân sách',
      INSUFFICIENT: 'Không đủ ngân sách',
      RESERVED: 'Đã giữ ngân sách',
      COMMITTED: 'Đã cam kết',
    }[result];
  });
  readonly timelineItems = computed<TimelineViewEvent[]>(() =>
    this.timeline().map((event) => ({
      ...event,
      label: timelineLabels[event.eventType] ?? event.eventType,
      actorRoleLabel:
        event.actorRoles
          .map((role) => timelineRoleLabels[role])
          .find((label) => label !== undefined) ?? 'Người dùng',
    })),
  );
  readonly isOwner = computed(() => this.request()?.requesterName === this.auth.username());
  readonly availableActions = computed<PurchaseRequestAction[]>(() => {
    const request = this.request();
    if (!request) {
      return [];
    }
    if (this.isOwner() && request.status === 'DRAFT') {
      return ['SUBMIT', 'CANCEL'];
    }
    if (this.isOwner() && request.status === 'CHANGES_REQUESTED') {
      return ['RESUBMIT', 'CANCEL'];
    }
    const roles = this.auth.roles();
    if (!this.isOwner() && request.status === 'SUBMITTED' && roles.includes('department_manager')) {
      return ['APPROVE', 'REQUEST_CHANGES', 'REJECT'];
    }
    if (!this.isOwner() && request.status === 'MANAGER_APPROVED' && roles.includes('finance')) {
      return ['APPROVE', 'REQUEST_CHANGES', 'REJECT'];
    }
    return [];
  });

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed()).subscribe((params) => {
      this.requestId.set(params.get('requestId') ?? '');
      this.request.set(null);
      this.selectedAction.set(null);
      this.transitionError.set(null);
      this.transitionSuccess.set(null);
      this.load();
      this.loadTimeline(1);
      this.loadBudgetCheck();
      this.loadAttachments();
      this.loadComments();
    });
  }

  retry(): void {
    this.load();
    this.loadBudgetCheck();
  }

  retryBudget(): void {
    this.loadBudgetCheck();
  }

  retryAttachments(): void {
    this.loadAttachments();
  }

  retryComments(): void {
    this.loadComments();
  }

  async addComment(): Promise<void> {
    const body = this.commentBody().trim();
    const requestId = this.requestId();
    if (!body || !requestId || body.length > 2000) {
      this.commentsError.set('Nội dung trao đổi phải có từ 1 đến 2.000 ký tự.');
      return;
    }
    this.submittingComment.set(true);
    this.commentsError.set(null);
    try {
      const comment = await firstValueFrom(this.procurement.addComment(requestId, body));
      this.comments.update((items) => [...items, comment]);
      this.commentBody.set('');
      this.loadTimeline(1);
    } catch (error: unknown) {
      this.commentsError.set(problemMessage(error, 'Không gửi được trao đổi. Hãy thử lại.'));
    } finally {
      this.submittingComment.set(false);
    }
  }

  selectAttachmentFile(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.selectedFile.set(input.files?.item(0) ?? null);
    this.attachmentError.set(null);
  }

  async uploadAttachment(fileInput: HTMLInputElement): Promise<void> {
    const file = this.selectedFile();
    const requestId = this.requestId();
    if (!file || !requestId) {
      this.attachmentError.set('Vui lòng chọn một tệp để tải lên.');
      return;
    }
    const rule = this.attachments();
    if (rule && file.size > rule.maxSizeBytes) {
      this.attachmentError.set('Tệp vượt quá giới hạn 10 MB.');
      return;
    }
    if (rule && !rule.allowedContentTypes.includes(file.type)) {
      this.attachmentError.set('Chỉ hỗ trợ PDF, DOCX, XLSX, JPG và PNG.');
      return;
    }
    this.uploadingAttachment.set(true);
    this.attachmentError.set(null);
    try {
      await firstValueFrom(
        this.procurement.uploadAttachment(requestId, this.selectedDocumentType(), file),
      );
      this.selectedFile.set(null);
      fileInput.value = '';
      this.loadAttachments();
      this.loadTimeline(1);
    } catch (error: unknown) {
      this.attachmentError.set(problemMessage(error, 'Không tải được tài liệu lên.'));
    } finally {
      this.uploadingAttachment.set(false);
    }
  }

  async downloadAttachment(attachmentId: string, fileName: string): Promise<void> {
    try {
      const content = await firstValueFrom(
        this.procurement.downloadAttachment(this.requestId(), attachmentId),
      );
      const url = URL.createObjectURL(content);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = fileName;
      anchor.click();
      URL.revokeObjectURL(url);
    } catch (error: unknown) {
      this.attachmentError.set(problemMessage(error, 'Không tải xuống được tài liệu.'));
    }
  }

  async deleteAttachment(attachmentId: string): Promise<void> {
    if (!this.requestId() || this.deletingAttachmentId()) {
      return;
    }
    this.deletingAttachmentId.set(attachmentId);
    this.attachmentError.set(null);
    try {
      await firstValueFrom(this.procurement.deleteAttachment(this.requestId(), attachmentId));
      this.loadAttachments();
      this.loadTimeline(1);
    } catch (error: unknown) {
      this.attachmentError.set(problemMessage(error, 'Không xóa được tài liệu.'));
    } finally {
      this.deletingAttachmentId.set(null);
    }
  }

  documentTypeLabel(type: AttachmentDocumentType): string {
    return {
      QUOTATION: 'Báo giá',
      SPECIFICATION: 'Đặc tả kỹ thuật',
      CONTRACT: 'Hợp đồng',
      OTHER: 'Tài liệu khác',
    }[type];
  }

  attachmentSize(bytes: number): string {
    if (bytes < 1024) {
      return `${bytes} B`;
    }
    if (bytes < 1024 * 1024) {
      return `${(bytes / 1024).toFixed(1)} KB`;
    }
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }

  chooseAction(action: PurchaseRequestAction): void {
    this.selectedAction.set(action);
    this.transitionComment.set('');
    this.transitionError.set(null);
    this.transitionSuccess.set(null);
  }

  cancelAction(): void {
    this.selectedAction.set(null);
    this.transitionComment.set('');
    this.transitionError.set(null);
  }

  actionLabel(action: PurchaseRequestAction): string {
    return {
      SUBMIT: 'Gửi duyệt',
      RESUBMIT: 'Gửi duyệt lại',
      CANCEL: 'Hủy phiếu',
      APPROVE: 'Phê duyệt',
      REJECT: 'Từ chối',
      REQUEST_CHANGES: 'Yêu cầu chỉnh sửa',
    }[action];
  }

  actionDescription(action: PurchaseRequestAction): string {
    return {
      SUBMIT: 'Phiếu sẽ được chuyển đến quản lý phòng ban.',
      RESUBMIT: 'Phiếu đã sửa sẽ quay lại vòng duyệt của quản lý.',
      CANCEL: 'Phiếu sẽ đóng và không thể tiếp tục quy trình.',
      APPROVE: 'Phiếu sẽ được chuyển sang bước tiếp theo của quy trình.',
      REJECT: 'Phiếu sẽ kết thúc với trạng thái bị từ chối.',
      REQUEST_CHANGES: 'Phiếu sẽ được trả lại cho người yêu cầu chỉnh sửa.',
    }[action];
  }

  requiresComment(action: PurchaseRequestAction | null): boolean {
    return action === 'REJECT' || action === 'REQUEST_CHANGES';
  }

  async performTransition(): Promise<void> {
    const request = this.request();
    const action = this.selectedAction();
    const comment = this.transitionComment().trim();
    if (!request || !action) {
      return;
    }
    if (this.requiresComment(action) && !comment) {
      this.transitionError.set('Vui lòng nhập lý do cho hành động này.');
      return;
    }

    this.transitioning.set(true);
    this.transitionError.set(null);
    try {
      const result = await firstValueFrom(
        this.procurement.transition(
          request.id,
          {
            action,
            expectedVersion: request.version,
            comment: comment || undefined,
          },
          crypto.randomUUID(),
        ),
      );
      this.request.set(result);
      this.transitionSuccess.set(
        `${this.actionLabel(action)} thành công. Phiếu hiện ở trạng thái ${result.status}.`,
      );
      this.selectedAction.set(null);
      this.transitionComment.set('');
      this.loadTimeline(1);
      this.loadBudgetCheck();
    } catch (error: unknown) {
      this.transitionError.set(
        problemMessage(error, 'Không thực hiện được hành động. Hãy tải lại phiếu và thử lại.'),
      );
    } finally {
      this.transitioning.set(false);
    }
  }

  goToTimelinePage(target: number): void {
    if (target < 1 || target > this.timelinePages() || target === this.timelinePage()) {
      return;
    }
    this.loadTimeline(target);
  }

  retryTimeline(): void {
    this.loadTimeline(this.timelinePage());
  }

  private load(): void {
    const requestId = this.requestId();
    if (!requestId) {
      this.error.set('Mã định danh phiếu không hợp lệ.');
      this.loading.set(false);
      return;
    }

    const generation = ++this.requestGeneration;
    this.loading.set(true);
    this.error.set(null);
    this.procurement
      .get(requestId)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (request) => {
          if (generation !== this.requestGeneration) {
            return;
          }
          this.request.set(request);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.requestGeneration) {
            return;
          }
          this.error.set(
            problemMessage(
              error,
              'Không tải được phiếu. Phiếu có thể không tồn tại hoặc nằm ngoài phạm vi của bạn.',
            ),
          );
          this.loading.set(false);
        },
      });
  }

  private loadTimeline(page: number): void {
    const requestId = this.requestId();
    if (!requestId) {
      return;
    }
    const generation = ++this.timelineGeneration;
    this.timelineLoading.set(true);
    this.timelineError.set(null);
    this.procurement
      .timeline(requestId, page, this.timelinePageSize)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (generation !== this.timelineGeneration) {
            return;
          }
          this.timeline.set(result.items);
          this.timelinePage.set(result.page);
          this.timelineTotal.set(result.total);
          this.timelinePages.set(result.pages);
          this.timelineLoading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.timelineGeneration) {
            return;
          }
          this.timelineError.set(problemMessage(error, 'Không tải được lịch sử xử lý của phiếu.'));
          this.timelineLoading.set(false);
        },
      });
  }

  private loadBudgetCheck(): void {
    const requestId = this.requestId();
    if (!requestId) {
      return;
    }
    const generation = ++this.budgetGeneration;
    this.budgetLoading.set(true);
    this.budgetError.set(null);
    this.procurement
      .budgetCheck(requestId)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (generation !== this.budgetGeneration) {
            return;
          }
          this.budget.set(result);
          this.budgetLoading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.budgetGeneration) {
            return;
          }
          this.budgetError.set(
            problemMessage(error, 'Không tải được thông tin kiểm tra ngân sách.'),
          );
          this.budgetLoading.set(false);
        },
      });
  }

  private loadAttachments(): void {
    const requestId = this.requestId();
    if (!requestId) {
      return;
    }
    const generation = ++this.attachmentGeneration;
    this.attachmentsLoading.set(true);
    this.attachmentError.set(null);
    this.procurement
      .attachments(requestId)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (generation !== this.attachmentGeneration) {
            return;
          }
          this.attachments.set(result);
          this.attachmentsLoading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.attachmentGeneration) {
            return;
          }
          this.attachmentError.set(problemMessage(error, 'Không tải được danh sách tài liệu.'));
          this.attachmentsLoading.set(false);
        },
      });
  }

  private loadComments(): void {
    const requestId = this.requestId();
    if (!requestId) {
      return;
    }
    const generation = ++this.commentGeneration;
    this.commentsLoading.set(true);
    this.commentsError.set(null);
    this.procurement
      .comments(requestId)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          if (generation !== this.commentGeneration) {
            return;
          }
          this.comments.set(result.items);
          this.commentsLoading.set(false);
        },
        error: (error: unknown) => {
          if (generation !== this.commentGeneration) {
            return;
          }
          this.commentsError.set(problemMessage(error, 'Không tải được nội dung trao đổi.'));
          this.commentsLoading.set(false);
        },
      });
  }
}
