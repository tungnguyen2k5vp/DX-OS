import { HttpErrorResponse } from '@angular/common/http';
import { ProblemDetails } from './procurement.models';

export function problemMessage(error: unknown, fallback: string): string {
  if (!(error instanceof HttpErrorResponse)) {
    return fallback;
  }

  const body = error.error;
  if (isProblemDetails(body)) {
    return body.detail || body.title || fallback;
  }
  if (error.status === 0) {
    return 'Không kết nối được dịch vụ xử lý dữ liệu. Hãy kiểm tra hệ thống và thử lại.';
  }
  return fallback;
}

export function problemViolations(error: unknown): Record<string, string> {
  if (!(error instanceof HttpErrorResponse) || !isProblemDetails(error.error)) {
    return {};
  }

  return Object.fromEntries(
    (error.error.errors ?? []).map((violation) => [violation.field, violation.message]),
  );
}

function isProblemDetails(value: unknown): value is ProblemDetails {
  if (!value || typeof value !== 'object') {
    return false;
  }
  const candidate = value as Partial<ProblemDetails>;
  return typeof candidate.status === 'number' && typeof candidate.code === 'string';
}
