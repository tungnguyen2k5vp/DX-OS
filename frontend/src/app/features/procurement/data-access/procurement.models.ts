export const purchaseRequestStatuses = [
  'DRAFT',
  'SUBMITTED',
  'MANAGER_APPROVED',
  'CHANGES_REQUESTED',
  'APPROVED',
  'REJECTED',
  'CANCELLED',
] as const;

export type PurchaseRequestStatus = (typeof purchaseRequestStatuses)[number];

export interface PurchaseRequestItem {
  id: string;
  lineNumber: number;
  description: string;
  quantity: string;
  unit: string;
  unitPrice: string;
  lineTotal: string;
}

export interface PurchaseRequest {
  id: string;
  requestCode: string;
  requesterId: string;
  requesterName: string;
  departmentId: string;
  departmentName: string;
  title: string;
  reason: string;
  currency: string;
  totalAmount: string;
  costCenter: string;
  status: PurchaseRequestStatus;
  version: number;
  createdAt: string;
  updatedAt: string;
  items?: PurchaseRequestItem[];
}

export interface PurchaseRequestPage {
  items: PurchaseRequest[];
  page: number;
  pageSize: number;
  total: number;
  pages: number;
}

export interface PurchaseRequestTimelineEvent {
  id: string;
  eventType: string;
  fromStatus: PurchaseRequestStatus | null;
  toStatus: PurchaseRequestStatus;
  actorName: string;
  actorRoles: string[];
  comment: string | null;
  occurredAt: string;
  correlationId: string | null;
}

export interface PurchaseRequestTimelinePage {
  items: PurchaseRequestTimelineEvent[];
  page: number;
  pageSize: number;
  total: number;
  pages: number;
}

export const attachmentDocumentTypes = ['QUOTATION', 'SPECIFICATION', 'CONTRACT', 'OTHER'] as const;

export type AttachmentDocumentType = (typeof attachmentDocumentTypes)[number];

export interface PurchaseRequestAttachment {
  id: string;
  purchaseRequestId: string;
  documentType: AttachmentDocumentType;
  fileName: string;
  contentType: string;
  sizeBytes: number;
  checksumSha256: string;
  uploadedBy: string;
  uploadedByName: string;
  uploadedAt: string;
}

export interface PurchaseRequestAttachmentList {
  items: PurchaseRequestAttachment[];
  required: boolean;
  requirementMet: boolean;
  requiredDocumentType?: AttachmentDocumentType;
  thresholdAmount?: string;
  maxSizeBytes: number;
  allowedContentTypes: string[];
}

export type BudgetCheckResult =
  'NOT_CONFIGURED' | 'AVAILABLE' | 'INSUFFICIENT' | 'RESERVED' | 'COMMITTED';

export type BudgetReservationState = 'RESERVED' | 'COMMITTED';

export interface BudgetSummary {
  periodCode: string;
  periodStart: string;
  periodEnd: string;
  costCenter: string;
  currency: string;
  allocatedAmount: string;
  reservedAmount: string;
  committedAmount: string;
  availableAmount: string;
}

export interface BudgetCheck {
  configured: boolean;
  result: BudgetCheckResult;
  requestedAmount: string;
  reservationState: BudgetReservationState | null;
  summary: BudgetSummary | null;
}

export type BudgetAlertLevel = 'HEALTHY' | 'WARNING' | 'CRITICAL';
export type BudgetReservationStatus = 'RESERVED' | 'COMMITTED' | 'RELEASED';

export interface BudgetAllocation {
  id: string;
  periodCode: string;
  periodStart: string;
  periodEnd: string;
  costCenter: string;
  currency: string;
  allocatedAmount: string;
  reservedAmount: string;
  committedAmount: string;
  availableAmount: string;
  utilization: string;
  alertLevel: BudgetAlertLevel;
  version: number;
}

export interface BudgetCurrencyTotal {
  currency: string;
  allocatedAmount: string;
  reservedAmount: string;
  committedAmount: string;
  availableAmount: string;
}

export interface BudgetReservation {
  id: string;
  purchaseRequestId: string;
  requestCode: string;
  requestTitle: string;
  costCenter: string;
  currency: string;
  amount: string;
  status: BudgetReservationStatus;
  reservedBy: string;
  reservedAt: string;
  committedAt: string | null;
  releasedAt: string | null;
}

export interface BudgetAdjustment {
  id: string;
  allocationId: string;
  costCenter: string;
  currency: string;
  previousAmount: string;
  adjustedAmount: string;
  reason: string;
  actorName: string;
  createdAt: string;
}

export interface BudgetDashboard {
  allocations: BudgetAllocation[];
  totals: BudgetCurrencyTotal[];
  reservations: BudgetReservation[];
  adjustments: BudgetAdjustment[];
  alertCount: number;
  canManage: boolean;
}

export interface AdjustBudgetAllocation {
  allocatedAmount: string;
  expectedVersion: number;
  reason: string;
}

export interface CreatePurchaseRequestItem {
  description: string;
  quantity: string;
  unit: string;
  unitPrice: string;
}

export interface CreatePurchaseRequest {
  title: string;
  reason: string;
  currency: string;
  costCenter: string;
  items: CreatePurchaseRequestItem[];
}

export interface UpdatePurchaseRequest extends CreatePurchaseRequest {
  expectedVersion: number;
}

export const purchaseRequestActions = [
  'SUBMIT',
  'RESUBMIT',
  'CANCEL',
  'APPROVE',
  'REJECT',
  'REQUEST_CHANGES',
] as const;

export type PurchaseRequestAction = (typeof purchaseRequestActions)[number];

export interface TransitionPurchaseRequest {
  action: PurchaseRequestAction;
  expectedVersion: number;
  comment?: string;
}

export interface ProblemFieldViolation {
  field: string;
  message: string;
}

export interface ProblemDetails {
  type: string;
  title: string;
  status: number;
  detail: string;
  instance: string;
  code: string;
  correlationId: string;
  errors?: ProblemFieldViolation[];
}

export function isPurchaseRequestStatus(value: string): value is PurchaseRequestStatus {
  return purchaseRequestStatuses.some((status) => status === value);
}
