import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { APP_CONFIG } from '../../../core/config/app-config';
import { AIDecision, AIRecommendation, AIRecommendationList } from './ai.models';

@Injectable({ providedIn: 'root' })
export class AIService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${inject(APP_CONFIG).apiBaseUrl}/api/v1/ai/recommendations`;

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
