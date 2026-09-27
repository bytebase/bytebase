import { unknownUser } from "@/types";
import type { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { appStoreUtilBridge } from "@/utils/app-store-bridge";

// Non-hook helpers that read the app store (the single source of truth) via
// the util bridge.

export const getCurrentUserV1 = (): User => {
  return appStoreUtilBridge()?.currentUser() ?? unknownUser();
};

export const hasFeature = (feature: PlanFeature): boolean => {
  return appStoreUtilBridge()?.hasFeature(feature) ?? false;
};
