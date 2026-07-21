import { unknownUser } from "@/types";
import type { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { appStoreUtilBridge } from "@/utils/app-store-bridge";

// Pinia-free replacements for helpers that used to live in the deleted legacy
// Pinia stores (`v1/auth`, `v1/subscription`). They read the React app store
// (the single source of truth) via the util bridge — the old Pinia stores were
// never populated in the React shell, so the previous implementations silently
// returned unknownUser() / no features.

export const getCurrentUserV1 = (): User => {
  return appStoreUtilBridge()?.currentUser() ?? unknownUser();
};

export const hasFeature = (feature: PlanFeature): boolean => {
  return appStoreUtilBridge()?.hasFeature(feature) ?? false;
};
