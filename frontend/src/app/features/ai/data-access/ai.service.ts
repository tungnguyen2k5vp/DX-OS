import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { APP_CONFIG } from '../../../core/config/app-config';
import {
  AIAssistantAnswer,
  AIAssistantStatus,
  AIDecision,
  AIRecommendation,
  AIRecommendationList,
} from './ai.models';

@Injectable({ providedIn: 'root' })
export class AIService {
  private readonly http = inject(HttpClient);
  private readonly aiBaseUrl = `${inject(APP_CONFIG).apiBaseUrl}/api/v1/ai`;
  private readonly baseUrl = `${this.aiBaseUrl}/recommendations`;

  assistantStatus(): Observable<AIAssistantStatus> {
    return this.http.get<AIAssistantStatus>(`${this.aiBaseUrl}/assistant/status`);
  }

  ask(question: string): Observable<AIAssistantAnswer> {
    return this.http.post<AIAssistantAnswer>(`${this.aiBaseUrl}/assistant/questions`, { question });
  }

  list(): Observable<AIRecommendationList> {
    return this.http.get<AIRecommendationList>(this.baseUrl);
  }

  generate(): Observable<AIRecommendationList> {
    return this.http.post<AIRecommendationList>(`${this.baseUrl}/generate`, {});
  }

  decide(id: string, input: AIDecision): Observable<AIRecommendation> {
    return this.http.post<AIRecommendation>(
      `${this.baseUrl}/${encodeURIComponent(id)}/decisions`,
      input,
    );
  }
}
