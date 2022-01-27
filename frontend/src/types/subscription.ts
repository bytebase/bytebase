import { PlanType } from "./plan";

export interface Subscription {
  instanceCount: number;
  expiresTs: number;
  plan: PlanType;
}

export interface SubscriptionState {
  subscription: Subscription | undefined;
}
