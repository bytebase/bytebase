import { PlanType } from "./plan";

export interface Subscription {
  instanceCount: number;
  expiresTs: number;
  startedTs: number;
  plan: PlanType;
  trialing: boolean;
}

export interface SubscriptionState {
  subscription: Subscription | undefined;
  trialingDays: number;
  trialingPlan: PlanType;
  trialingInstanceCount: number;
}
