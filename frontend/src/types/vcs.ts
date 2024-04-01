import type { VCSType } from "@/types/proto/v1/common";

// When configuring the VCS, we split the SaaS and self-hosted into two types to present optimal UX.
export type VCSUIType =
  | "GITLAB_SELF_HOST"
  | "GITLAB_COM"
  | "GITHUB_COM"
  | "GITHUB_ENTERPRISE"
  | "BITBUCKET_ORG"
  | "AZURE_DEVOPS";

export interface VCSConfig {
  type: VCSType;
  uiType: VCSUIType;
  resourceId: string;
  name: string;
  instanceUrl: string;
  accessToken: string;
}
