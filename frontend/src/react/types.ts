export interface SubscriptionData {
  currentPlan: string;
  planType: "FREE" | "TEAM" | "ENTERPRISE";
  isFreePlan: boolean;
  isTrialing: boolean;
  isExpired: boolean;
  isSelfHostLicense: boolean;
  showTrial: boolean;
  trialingDays: number;
  expireAt: string;
  instanceCountLimit: number;
  instanceLicenseCount: number;
  userCountLimit: number;
  activeUserCount: number;
  activatedInstanceCount: number;
  workspaceId: string;
}
