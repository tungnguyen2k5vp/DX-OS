import { PurchaseRequest } from '../data-access/procurement.models';

export function isPurchaseRequestOwner(
  request: PurchaseRequest | null | undefined,
  authenticatedUsername: string,
): boolean {
  const username = authenticatedUsername.trim();
  return Boolean(request && username && request.requesterUsername === username);
}
