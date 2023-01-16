import { PlanType } from "./plan";

export interface Subscription {
  expiresTs: number;
  startedTs: number;
  plan: PlanType;
  trialing: boolean;
}

export interface SubscriptionState {
  subscription: Subscription | undefined;
  trialingDays: number;
}
