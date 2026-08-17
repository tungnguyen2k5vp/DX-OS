export type AIRecommendationStatus = 'PENDING' | 'APPROVED' | 'REJECTED' | 'DISMISSED';
export type AIRiskLevel = 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL';

export interface AIRecommendation {
  id: string;
  purchaseRequestId?: string;
  purchaseRequestCode?: string;
  purchaseRequestTitle?: string;
  recommendationType: string;
  title: string;
  summary: string;
  riskLevel: AIRiskLevel;
  evidence: Record<string, unknown>;
  status: AIRecommendationStatus;
  generatedBy: string;
  generatedAt: string;
  decidedBy?: string;
  decidedAt?: string;
  decisionComment?: string;
  version: number;
}

export interface AIRecommendationList {
  items: AIRecommendation[];
  total: number;
  pending: number;
  highRisk: number;
  canOperate: boolean;
  methodology: string;
}

export interface AIDecision {
  status: Exclude<AIRecommendationStatus, 'PENDING'>;
  comment: string;
  expectedVersion: number;
}
