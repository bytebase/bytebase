import { FeatureType } from "./plan";
import { Subscription } from "@/types/proto/v1/subscription_service";

export interface SubscriptionState {
  featureMatrix: Map<FeatureType, boolean[]>;
  subscription: Subscription | undefined;
  trialingDays: number;
}
