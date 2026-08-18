import { DatePipe, JsonPipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { RouterLink } from '@angular/router';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { problemMessage } from '../../../procurement/data-access/problem-details';
import {
  AIRecommendation,
  AIRecommendationList,
  AIRecommendationStatus,
  AIRiskLevel,
} from '../../data-access/ai.models';
import { AIService } from '../../data-access/ai.service';

@Component({
  selector: 'app-recommendation-center',
  imports: [DatePipe, JsonPipe, RouterLink, HlmBadge, HlmButton, ...HlmCardImports],
  templateUrl: './recommendation-center.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RecommendationCenterPage {
  private readonly service = inject(AIService);
  private readonly destroyRef = inject(DestroyRef);

  readonly result = signal<AIRecommendationList | null>(null);
  readonly loading = signal(true);
  readonly busy = signal(false);
  readonly error = signal<string | null>(null);
  readonly comments = signal<Record<string, string>>({});

  constructor() {
    this.load();
  }

  generate(): void {
    this.busy.set(true);
    this.error.set(null);
    this.service
      .generate()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          this.result.set(result);
          this.busy.set(false);
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không tạo được khuyến nghị.'));
          this.busy.set(false);
        },
      });
  }

  updateComment(id: string, value: string): void {
    this.comments.update((current) => ({ ...current, [id]: value }));
  }

  riskLabel(risk: AIRiskLevel): string {
    return {
      LOW: 'Thấp',
      MEDIUM: 'Trung bình',
      HIGH: 'Cao',
      CRITICAL: 'Nghiêm trọng',
    }[risk];
  }

  statusLabel(status: AIRecommendationStatus): string {
    return {
      PENDING: 'Chờ quyết định',
      APPROVED: 'Đã chấp thuận',
      REJECTED: 'Đã từ chối',
      DISMISSED: 'Đã bỏ qua',
    }[status];
  }

  decide(item: AIRecommendation, status: Exclude<AIRecommendationStatus, 'PENDING'>): void {
    const comment = this.comments()[item.id]?.trim() || '';
    if (comment.length < 5) {
      this.error.set('Nhập lý do quyết định ít nhất 5 ký tự.');
      return;
    }
    this.busy.set(true);
    this.error.set(null);
    this.service
      .decide(item.id, { status, comment, expectedVersion: item.version })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.busy.set(false);
          this.load();
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không lưu được quyết định.'));
          this.busy.set(false);
        },
      });
  }

  private load(): void {
    this.loading.set(true);
    this.service
      .list()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (result) => {
          this.result.set(result);
          this.loading.set(false);
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không tải được trung tâm khuyến nghị.'));
          this.loading.set(false);
        },
      });
  }
}
