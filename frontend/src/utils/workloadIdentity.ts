import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";

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

  const providerType = wi.workloadIdentityConfig.providerType;
  switch (providerType) {
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
  providerType: WorkloadIdentityConfig_ProviderType
) => {
  switch (providerType) {
    case WorkloadIdentityConfig_ProviderType.GITHUB:
      return "GitHub Actions";
    case WorkloadIdentityConfig_ProviderType.GITLAB:
      return "GitLab CI";
    default:
      return "";
  }
};
