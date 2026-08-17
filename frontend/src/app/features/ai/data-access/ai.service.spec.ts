import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { APP_CONFIG } from '../../../core/config/app-config';
import { AIService } from './ai.service';

describe('AIService', () => {
  it('generates and decides explainable recommendations', () => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: APP_CONFIG, useValue: { apiBaseUrl: 'http://api.test' } },
      ],
    });
    const service = TestBed.inject(AIService);
    const http = TestBed.inject(HttpTestingController);
    service.generate().subscribe();
    const generate = http.expectOne('http://api.test/api/v1/ai/recommendations/generate');
    expect(generate.request.method).toBe('POST');
    generate.flush({ items: [], total: 0 });

    service
      .decide('recommendation-id', {
        status: 'APPROVED',
        comment: 'Evidence reviewed.',
        expectedVersion: 1,
      })
      .subscribe();
    const decide = http.expectOne(
      'http://api.test/api/v1/ai/recommendations/recommendation-id/decisions',
    );
    expect(decide.request.body.expectedVersion).toBe(1);
    decide.flush({ id: 'recommendation-id', status: 'APPROVED' });
    http.verify();
  });
});
