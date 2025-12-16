export interface SpannerChildLink {
  childIndex: number;
  type?: string;
  variable?: string;
}

export interface SpannerShortRepresentation {
  description: string;
  subqueries?: Record<string, number>;
}

export interface SpannerPlanNodeData {
  index: number;
  kind: string;
  displayName: string;
  childLinks?: SpannerChildLink[];
  shortRepresentation?: SpannerShortRepresentation;
  metadata?: Record<string, unknown>;
  executionStats?: Record<string, unknown>;
}

export interface SpannerQueryPlanData {
  planNodes: SpannerPlanNodeData[];
}
