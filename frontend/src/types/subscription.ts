import { PlanType, FeatureType } from "./plan";

export interface Subscription {
  instanceCount: number;
  seat: number;
  expiresTs: number;
  startedTs: number;
  plan: PlanType;
  trialing: boolean;
}

export interface SubscriptionState {
  featureMatrix: Map<FeatureType, boolean[]>;
  subscription: Subscription | undefined;
  trialingDays: number;
}
