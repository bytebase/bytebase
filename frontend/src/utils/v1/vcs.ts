import { VCSUIType } from "@/types";
import {
  VCSProvider,
  VCSProvider_Type,
} from "@/types/proto/v1/vcs_provider_service";
import { isDev } from "@/utils";

export const getVCSUIType = (vcs: VCSProvider): VCSUIType => {
  switch (vcs.type) {
    case VCSProvider_Type.GITHUB:
      return "GITHUB_COM";
    case VCSProvider_Type.BITBUCKET:
      return "BITBUCKET_ORG";
    case VCSProvider_Type.GITLAB:
      if (vcs.url === "https://gitlab.com") {
        return "GITLAB_COM";
      }
      return "GITLAB_SELF_HOST";
    case VCSProvider_Type.AZURE_DEVOPS:
      return "AZURE_DEVOPS";
    default:
      return "GITLAB_SELF_HOST";
  }
};

export const supportSQLReviewCI = (vcsType: VCSProvider_Type): boolean => {
  return (
    vcsType === VCSProvider_Type.GITHUB ||
    vcsType === VCSProvider_Type.GITLAB ||
    vcsType === VCSProvider_Type.AZURE_DEVOPS ||
    (vcsType === VCSProvider_Type.BITBUCKET && isDev())
  );
};
