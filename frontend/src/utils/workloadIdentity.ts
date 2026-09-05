import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";

// The audience the GitOps page's generated workflows request, and the one the
// create form presets. An identity and the pipeline that authenticates to it
// must name the same audience, so both read it here.
export const GENERATED_WORKFLOW_AUDIENCE = "bytebase";

// provider_type is optional in storage and identities written before it was
// required may carry PROVIDER_TYPE_UNSPECIFIED. Every reader that needs a
// provider for such an identity resolves it here, or two readers disagree

// Parse subject pattern and extract owner/repo/branch/refType
export const parseWorkloadIdentitySubjectPattern = (wi: {
  workloadIdentityConfig?: {
    subjectPattern: string;
    providerType: WorkloadIdentityConfig_ProviderType;
  };
}) => {
  if (!wi.workloadIdentityConfig) {
    return;
  }

  const pattern = wi.workloadIdentityConfig.subjectPattern;
  if (!pattern) {
    return;
  }

  switch (wi.workloadIdentityConfig.providerType) {
    case WorkloadIdentityConfig_ProviderType.GITHUB: {
      const match = /^repo:([^/]+)\/(.*)$/.exec(pattern);
      if (!match) return;
      const owner = match[1];
      const rest = match[2];
      if (rest === "*") return { owner, repo: "", branch: "" };
      const repoMatch = /^([^:]+):(.*)$/.exec(rest);
      if (!repoMatch) return;
      const repo = repoMatch[1];
      const refPart = repoMatch[2];
      if (refPart === "*") return { owner, repo, branch: "" };
      const branchMatch = /^ref:refs\/heads\/(.+)$/.exec(refPart);
      return { owner, repo, branch: branchMatch?.[1] ?? "" };
    }
    case WorkloadIdentityConfig_ProviderType.GITLAB: {
      const match = /^project_path:([^/]+)\/(.*)$/.exec(pattern);
      if (!match) return;
      const owner = match[1];
      const rest = match[2];
      if (rest === "*") return { owner, repo: "", branch: "" };
      const projectMatch = /^([^:]+):(.*)$/.exec(rest);
      if (!projectMatch) return;
      const repo = projectMatch[1];
      const refPart = projectMatch[2];
      if (refPart === "*") return { owner, repo, branch: "" };
      const refTypeMatch = /^ref_type:(branch|tag):ref:(.+)$/.exec(refPart);
      return {
        owner,
        repo,
        branch: refTypeMatch?.[2] ?? "",
        refType: refTypeMatch?.[1] as "branch" | "tag",
      };
    }
    default:
      return;
  }
};

export const getWorkloadIdentityProviderText = (
  providerType: WorkloadIdentityConfig_ProviderType,
  genericOIDCText = ""
) => {
  switch (providerType) {
    case WorkloadIdentityConfig_ProviderType.GITHUB:
      return "GitHub Actions";
    case WorkloadIdentityConfig_ProviderType.GITLAB:
      return "GitLab CI";
    case WorkloadIdentityConfig_ProviderType.OIDC:
      return genericOIDCText;
    default:
      return "";
  }
};
