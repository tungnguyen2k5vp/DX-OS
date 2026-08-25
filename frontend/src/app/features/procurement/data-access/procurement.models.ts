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
  requesterUsername: string;
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

export interface PurchaseRequestComment {
  id: string;
  body: string;
  authorId: string;
  authorName: string;
  authorRoles: string[];
  createdAt: string;
}

export interface PurchaseRequestCommentList {
  items: PurchaseRequestComment[];
  total: number;
}

export type WorkTaskType = 'COMPLETE_REQUEST' | 'MANAGER_REVIEW' | 'FINANCE_REVIEW' | 'SLA_MONITOR';

export type WorkTaskUrgency = 'NORMAL' | 'DUE_SOON' | 'OVERDUE';

export interface WorkTask {
  purchaseRequestId: string;
  requestCode: string;
  title: string;
  requesterName: string;
  departmentName: string;
  status: PurchaseRequestStatus;
  taskType: WorkTaskType;
  currency: string;
  totalAmount: string;
  dueAt: string | null;
  overdue: boolean;
  urgency: WorkTaskUrgency;
  updatedAt: string;
}

export interface WorkSummary {
  items: WorkTask[];
  total: number;
  overdueCount: number;
  dueSoonCount: number;
}

export type SupplierStatus = 'ACTIVE' | 'INACTIVE';
export type SupplierRiskLevel = 'LOW' | 'MEDIUM' | 'HIGH';
export type SupplierComplianceStatus = 'PENDING' | 'VERIFIED' | 'EXPIRED' | 'BLOCKED';

export interface Supplier {
  id: string;
  code: string;
  name: string;
  taxCode?: string;
  contactName?: string;
  email?: string;
  phone?: string;
  address?: string;
  bankName?: string;
  bankAccountNumber?: string;
  contractReference?: string;
  contractExpiresOn?: string;
  complianceStatus: SupplierComplianceStatus;
  performanceScore?: string;
  businessNote?: string;
  status: SupplierStatus;
  riskLevel: SupplierRiskLevel;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface SupplierInput {
  code: string;
  name: string;
  taxCode: string;
  contactName: string;
  email: string;
  phone: string;
  address: string;
  bankName: string;
  bankAccountNumber: string;
  contractReference: string;
  contractExpiresOn: string;
  complianceStatus: SupplierComplianceStatus;
  performanceScore: string;
  businessNote: string;
  status: SupplierStatus;
  riskLevel: SupplierRiskLevel;
  expectedVersion?: number;
}

export interface SupplierList {
  items: Supplier[];
  total: number;
  canManage: boolean;
}

export type FulfillmentStatus =
  | 'AWAITING_ORDER'
  | 'ORDERED'
  | 'PARTIALLY_RECEIVED'
  | 'RECEIPT_EXCEPTION'
  | 'RECEIVED'
  | 'CANCELLED';

export interface PurchaseOrder {
  id: string;
  purchaseRequestId: string;
  requestCode: string;
  requestTitle: string;
  requesterName: string;
  departmentName: string;
  currency: string;
  totalAmount: string;
  orderCode: string | null;
  supplierId: string | null;
  supplierCode: string | null;
  supplierName: string | null;
  externalReference: string | null;
  expectedDeliveryOn: string | null;
  actualDeliveryOn: string | null;
  status: FulfillmentStatus;
  note: string | null;
  version: number;
  orderedAt: string | null;
  receivedAt: string | null;
  cancelledAt: string | null;
  cancellationReason: string | null;
  receiptCount: number;
  deliveryOverdue: boolean;
  canPlaceOrder: boolean;
  canConfirmReceipt: boolean;
  canManageOrder: boolean;
}

export interface OperationsBoard {
  items: PurchaseOrder[];
  total: number;
  awaitingOrderCount: number;
  inDeliveryCount: number;
  overdueDeliveryCount: number;
  receivedCount: number;
  partialCount: number;
  exceptionCount: number;
  cancelledCount: number;
}

export interface CreatePurchaseOrder {
  purchaseRequestId: string;
  supplierId: string;
  externalReference: string;
  expectedDeliveryOn: string;
  note: string;
}

export type ReceiptOutcome = 'PARTIAL' | 'COMPLETE' | 'DAMAGED' | 'WRONG_ITEM' | 'REJECTED';
export type ReceiptCondition = 'ACCEPTED' | 'DAMAGED' | 'WRONG_ITEM' | 'REJECTED';

export interface RecordReceiptItem {
  purchaseRequestItemId: string;
  quantityReceived: string;
  condition: ReceiptCondition;
  note: string;
}

export interface RecordReceipt {
  expectedVersion: number;
  outcome: ReceiptOutcome;
  receivedOn: string;
  note: string;
  items: RecordReceiptItem[];
}

export interface ReceiptItem extends RecordReceiptItem {
  lineNumber: number;
  description: string;
  orderedQuantity: string;
}

export interface ReceiptRecord {
  id: string;
  receiptNumber: string;
  outcome: ReceiptOutcome;
  receivedOn: string;
  note: string;
  createdBy: string;
  createdAt: string;
  items: ReceiptItem[];
}

export interface ReceiptHistory {
  items: ReceiptRecord[];
  total: number;
}

export interface UpdatePurchaseOrder extends Omit<CreatePurchaseOrder, 'purchaseRequestId'> {
  expectedVersion: number;
}

export type InvoiceStatus = 'RECORDED' | 'VERIFIED' | 'DISPUTED' | 'PAID';
export type InvoiceMatchStatus =
  | 'NOT_RECORDED'
  | 'WAITING_RECEIPT'
  | 'CURRENCY_MISMATCH'
  | 'AMOUNT_MISMATCH'
  | 'PARTIAL_MATCH'
  | 'MATCHED';

export interface InvoiceBoardItem {
  purchaseOrderId: string;
  purchaseRequestId: string;
  requestCode: string;
  requestTitle: string;
  requesterName: string;
  departmentName: string;
  supplierId: string;
  supplierCode: string;
  supplierName: string;
  orderCode: string;
  orderStatus: FulfillmentStatus;
  orderAmount: string;
  orderCurrency: string;
  actualDeliveryOn: string | null;
  invoiceId: string | null;
  invoiceNumber: string | null;
  issuedOn: string | null;
  dueOn: string | null;
  invoiceAmount: string | null;
  invoiceCurrency: string | null;
  invoiceStatus: InvoiceStatus | null;
  matchStatus: InvoiceMatchStatus;
  note: string | null;
  version: number;
  paymentReference: string | null;
  paidOn: string | null;
  paidAmount: string;
  remainingAmount: string;
  paymentCount: number;
  invoiceCreatedAt: string | null;
  invoiceUpdatedAt: string | null;
  paymentOverdue: boolean;
  canManage: boolean;
}

export interface InvoiceBoard {
  items: InvoiceBoardItem[];
  total: number;
  awaitingInvoiceCount: number;
  needsReviewCount: number;
  readyToPayCount: number;
  overdueCount: number;
  paidCount: number;
  canManage: boolean;
}

export interface CreateInvoice {
  purchaseOrderId: string;
  invoiceNumber: string;
  issuedOn: string;
  dueOn: string;
  amount: string;
  currency: string;
  note: string;
}

export interface UpdateInvoice extends Omit<CreateInvoice, 'purchaseOrderId'> {
  expectedVersion: number;
}

export type InvoiceAction = 'VERIFY' | 'DISPUTE' | 'REOPEN' | 'MARK_PAID';

export interface TransitionInvoice {
  action: InvoiceAction;
  expectedVersion: number;
  comment?: string;
  paymentReference?: string;
  paidOn?: string;
}

export interface RecordInvoicePayment {
  expectedVersion: number;
  amount: string;
  paidOn: string;
  paymentReference: string;
  note: string;
}

export interface InvoicePayment {
  id: string;
  amount: string;
  paidOn: string;
  paymentReference: string;
  note: string;
  createdBy: string;
  createdAt: string;
}

export interface InvoicePaymentList {
  items: InvoicePayment[];
  total: number;
}

export interface SLAPolicy {
  processName: string;
  targetHours: number;
  active: boolean;
  version: number;
}

export interface AttachmentPolicy {
  id: string;
  currency: string;
  thresholdAmount: string;
  requiredDocumentType: AttachmentDocumentType;
  active: boolean;
  version: number;
}

export interface PolicyCenter {
  slaPolicies: SLAPolicy[];
  attachmentRules: AttachmentPolicy[];
  canManage: boolean;
}

export interface UpdateSLAPolicy {
  targetHours: number;
  active: boolean;
  expectedVersion: number;
}

export interface UpdateAttachmentPolicy {
  thresholdAmount: string;
  requiredDocumentType: AttachmentDocumentType;
  active: boolean;
  expectedVersion: number;
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

export interface AddPurchaseRequestComment {
  body: string;
}

export interface ProcurementCatalogItem {
  id: string;
  code: string;
  name: string;
  description: string;
  category: string;
  unit: string;
  referenceUnitPrice: string;
  currency: string;
}

export interface ProcurementCatalog {
  items: ProcurementCatalogItem[];
  total: number;
}

export interface DuplicateRequestCandidate {
  purchaseRequestId: string;
  requestCode: string;
  title: string;
  status: PurchaseRequestStatus;
  totalAmount: string;
  currency: string;
  similarity: number;
  reason: string;
}

export interface DuplicateRequestCheck {
  potentialDuplicate: boolean;
  items: DuplicateRequestCandidate[];
}

export interface DuplicateRequestInput {
  title: string;
  costCenter: string;
  totalAmount: string;
  excludeRequestId?: string;
}

export interface ApprovalRule {
  id: string;
  departmentId?: string;
  departmentName?: string;
  name: string;
  currency: string;
  minimumAmount: string;
  maximumAmount?: string;
  requiresManager: boolean;
  requiresFinance: boolean;
  priority: number;
  active: boolean;
  version: number;
}

export interface ApprovalRuleInput {
  departmentId: string;
  name: string;
  currency: string;
  minimumAmount: string;
  maximumAmount: string;
  requiresManager: boolean;
  requiresFinance: boolean;
  priority: number;
  active: boolean;
  expectedVersion: number;
}

export interface ApprovalDelegation {
  id: string;
  departmentId: string;
  departmentName: string;
  delegatorUserId: string;
  delegatorName: string;
  delegateUserId: string;
  delegateName: string;
  delegateRoles: string[];
  startsOn: string;
  endsOn: string;
  reason: string;
  active: boolean;
  currentlyEffective: boolean;
  version: number;
}

export interface ApprovalDelegateCandidate {
  id: string;
  username: string;
  displayName: string;
  departmentName: string;
  roles: string[];
}

export interface ApprovalGovernance {
  rules: ApprovalRule[];
  delegations: ApprovalDelegation[];
  delegateCandidates: ApprovalDelegateCandidate[];
  canManageRules: boolean;
  canDelegate: boolean;
}

export interface CreateApprovalDelegation {
  delegateUserId: string;
  startsOn: string;
  endsOn: string;
  reason: string;
}

export interface SupplierQuote {
  id: string;
  sourcingCaseId: string;
  supplierId: string;
  supplierCode: string;
  supplierName: string;
  quoteReference: string;
  amount: string;
  currency: string;
  deliveryOn: string;
  warrantyMonths: number;
  paymentTerms: string;
  note?: string;
  status: 'SUBMITTED' | 'SELECTED' | 'REJECTED';
  priceScore: number;
  deliveryScore: number;
  qualityScore: number;
  complianceScore: number;
  totalScore: number;
  recommendation: string;
  version: number;
}

export interface SourcingCase {
  id?: string;
  purchaseRequestId: string;
  requestCode: string;
  requestTitle: string;
  departmentName: string;
  requesterName: string;
  requestAmount: string;
  currency: string;
  status: 'NOT_STARTED' | 'OPEN' | 'AWARDED';
  selectedQuoteId?: string;
  quotes: SupplierQuote[];
  recommendedQuoteId?: string;
  potentialSavings: string;
  canManage: boolean;
  version: number;
}

export interface SourcingBoard {
  items: SourcingCase[];
  total: number;
  awaitingQuotes: number;
  inComparison: number;
  awarded: number;
  canManage: boolean;
}

export interface SupplierQuoteInput {
  purchaseRequestId: string;
  supplierId: string;
  quoteReference: string;
  amount: string;
  currency: string;
  deliveryOn: string;
  warrantyMonths: number;
  paymentTerms: string;
  note: string;
  expectedVersion?: number;
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
