import { describe, expect, it } from 'vitest';
import { PurchaseRequest } from '../data-access/procurement.models';
import { isPurchaseRequestOwner } from './purchase-request-ownership';

const request = {
  requesterUsername: 'employee.demo',
  requesterName: 'Nguyễn Minh Anh',
} as PurchaseRequest;

describe('isPurchaseRequestOwner', () => {
  it('recognizes the owner when the display name differs from the login username', () => {
    expect(isPurchaseRequestOwner(request, 'employee.demo')).toBe(true);
  });

  it('does not use the display name as an authorization identifier', () => {
    expect(isPurchaseRequestOwner(request, 'Nguyễn Minh Anh')).toBe(false);
  });

  it('rejects missing requests and blank usernames', () => {
    expect(isPurchaseRequestOwner(null, 'employee.demo')).toBe(false);
    expect(isPurchaseRequestOwner(request, '   ')).toBe(false);
  });
});
