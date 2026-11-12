import { useMemo } from "react";
import { useAppStore } from "@/stores/app";
import { getWorkloadIdentityNameInBinding } from "@/types";
import {
  type WorkloadIdentity,
  WorkloadIdentityConfig_ProviderType,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  isBindingPolicyExpired,
  parseWorkloadIdentitySubjectPattern,
} from "@/utils";
import type { WorkflowProvider } from "../gitopsWorkflowYaml";

// The git provider setup has workflow files for, or undefined for any other
// OIDC provider.
export function workflowProviderOf(
  identity: WorkloadIdentity | undefined
): WorkflowProvider | undefined {
  switch (identity?.workloadIdentityConfig?.providerType) {
    case WorkloadIdentityConfig_ProviderType.GITHUB:
      return "github";
    case WorkloadIdentityConfig_ProviderType.GITLAB:
      return "gitlab";
    default:
      return undefined;
  }
}

// The repositories an identity lets sign in: one repository, or every
// repository of an owner when its subject pattern names no repository.
export function repositoriesOf(identity: WorkloadIdentity): {
  owner: string;
  repo: string;
} {
  const parsed = parseWorkloadIdentitySubjectPattern(identity);
  return { owner: parsed?.owner ?? "", repo: parsed?.repo ?? "" };
}

// The web address of the one repository an identity names, or "" when it
// trusts every repository of an owner.
export function repositoryUrlOf(identity: WorkloadIdentity): string {
  const { owner, repo } = repositoriesOf(identity);
  if (!owner || !repo) return "";
  const provider = workflowProviderOf(identity);
  if (provider === "github") return `https://github.com/${owner}/${repo}`;
  if (provider === "gitlab") {
    const issuer =
      identity.workloadIdentityConfig?.issuerUrl || "https://gitlab.com";
    return `${issuer.replace(/\/$/, "")}/${owner}/${repo}`;
  }
  return "";
}

// The roles an identity holds in the project, from the project's bindings and
// the workspace's. Undefined while the project policy cannot be read, so the
// caller skips the role check instead of reporting a missing role.
export function useIdentityRoles(
  projectName: string,
  identity: WorkloadIdentity | undefined
): string[] | undefined {
  const projectPolicy = useAppStore(
    (state) => state.projectPoliciesByName[projectName]
  );
  const workspacePolicy = useAppStore((state) => state.workspacePolicy);
  return useMemo(() => {
    if (!identity || !projectPolicy) return undefined;
    const member = getWorkloadIdentityNameInBinding(identity.email);
    const roles = new Set<string>();
    for (const binding of [
      ...projectPolicy.bindings,
      ...(workspacePolicy?.bindings ?? []),
    ]) {
      if (isBindingPolicyExpired(binding)) continue;
      if (binding.members.includes(member)) roles.add(binding.role);
    }
    return [...roles];
  }, [identity, projectPolicy, workspacePolicy]);
}
