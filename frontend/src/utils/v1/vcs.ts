import type { VCSUIType } from "@/types";
import { VCSType } from "@/types/proto/v1/common";
import type { VCSProvider } from "@/types/proto/v1/vcs_provider_service";

export const getVCSUIType = (vcs: VCSProvider): VCSUIType => {
  switch (vcs.type) {
    case VCSType.GITHUB:
      return "GITHUB_COM";
    case VCSType.BITBUCKET:
      return "BITBUCKET_ORG";
    case VCSType.GITLAB:
      if (vcs.url === "https://gitlab.com") {
        return "GITLAB_COM";
      }
      return "GITLAB_SELF_HOST";
    case VCSType.AZURE_DEVOPS:
      return "AZURE_DEVOPS";
    default:
      return "GITLAB_SELF_HOST";
  }
};
