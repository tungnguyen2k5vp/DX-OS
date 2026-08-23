import { DatePipe } from '@angular/common';
import { ChangeDetectionStrategy, Component, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { HlmBadge } from '@spartan-ng/helm/badge';
import { HlmButton } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { problemMessage } from '../../../procurement/data-access/problem-details';
import { AIAssistantAnswer, AIAssistantStatus } from '../../data-access/ai.models';
import { AIService } from '../../data-access/ai.service';
import { AssistantMarkdownPipe } from './assistant-markdown.pipe';

@Component({
  selector: 'app-ai-assistant',
  imports: [DatePipe, AssistantMarkdownPipe, HlmBadge, HlmButton, ...HlmCardImports],
  templateUrl: './assistant.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AIAssistantPage {
  private readonly service = inject(AIService);
  private readonly destroyRef = inject(DestroyRef);

  readonly status = signal<AIAssistantStatus | null>(null);
  readonly answer = signal<AIAssistantAnswer | null>(null);
  readonly question = signal('');
  readonly loadingStatus = signal(true);
  readonly asking = signal(false);
  readonly error = signal<string | null>(null);
  readonly examples = [
    'Phiếu mua sắm từ 20 triệu cần những tài liệu gì?',
    'Quy trình phê duyệt phiếu mua sắm diễn ra như thế nào?',
    'Khi nào ngân sách được giữ và khi nào được hoàn lại?',
  ];

  constructor() {
    this.refreshStatus();
  }

  updateQuestion(value: string): void {
    this.question.set(value);
  }

  useExample(value: string): void {
    this.question.set(value);
    this.ask();
  }

  ask(): void {
    const question = this.question().trim();
    if (question.length < 3) {
      this.error.set('Nhập câu hỏi có ít nhất 3 ký tự.');
      return;
    }
    this.asking.set(true);
    this.answer.set(null);
    this.error.set(null);
    this.service
      .ask(question)
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (answer) => {
          this.answer.set(answer);
          this.asking.set(false);
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Trợ lý AI local chưa thể trả lời.'));
          this.asking.set(false);
          this.refreshStatus();
        },
      });
  }

  refreshStatus(): void {
    this.loadingStatus.set(true);
    this.service
      .assistantStatus()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (status) => {
          this.status.set(status);
          this.loadingStatus.set(false);
        },
        error: (error: unknown) => {
          this.error.set(problemMessage(error, 'Không kiểm tra được trạng thái AI local.'));
          this.loadingStatus.set(false);
        },
      });
  }
}
