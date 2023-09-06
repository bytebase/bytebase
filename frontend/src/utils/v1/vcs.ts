import { VCSUIType } from "@/types";
import {
  ExternalVersionControl,
  ExternalVersionControl_Type,
} from "@/types/proto/v1/externalvs_service";

export const getVCSUIType = (vcs: ExternalVersionControl): VCSUIType => {
  switch (vcs.type) {
    case ExternalVersionControl_Type.GITHUB:
      return "GITHUB_COM";
    case ExternalVersionControl_Type.BITBUCKET:
      return "BITBUCKET_ORG";
    case ExternalVersionControl_Type.GITLAB:
      if (vcs.url === "https://gitlab.com") {
        return "GITLAB_COM";
      }
      return "GITLAB_SELF_HOST";
    case ExternalVersionControl_Type.AZURE_DEVOPS:
      return "AZURE_DEVOPS";
    default:
      return "GITLAB_SELF_HOST";
  }
};
