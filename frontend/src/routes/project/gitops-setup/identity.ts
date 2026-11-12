import { useMemo } from "react";
import { useAppStore } from "@/stores/app";
import {
  ALL_USERS_USER_EMAIL,
  getWorkloadIdentityNameInBinding,
} from "@/types";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
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
// the workspace's. A binding counts when it names the identity or `allUsers`,
// as the server's permission check does: `allUsers` in a project covers every
// account, and `allUsers` in the workspace is skipped on Bytebase Cloud.
// Undefined while the project policy cannot be read, so the caller skips the
// role check instead of reporting a missing role.
export function useIdentityRoles(
  projectName: string,
  identity: WorkloadIdentity | undefined
): string[] | undefined {
  const projectPolicy = useAppStore(
    (state) => state.projectPoliciesByName[projectName]
  );
  const workspacePolicy = useAppStore((state) => state.workspacePolicy);
  const isSaaSMode = useAppStore((state) => state.isSaaSMode());
  return useMemo(() => {
    if (!identity || !projectPolicy) return undefined;
    const member = getWorkloadIdentityNameInBinding(identity.email);
    const roles = new Set<string>();
    const collect = (bindings: Binding[], allUsersCounts: boolean) => {
      for (const binding of bindings) {
        if (isBindingPolicyExpired(binding)) continue;
        if (
          binding.members.includes(member) ||
          (allUsersCounts && binding.members.includes(ALL_USERS_USER_EMAIL))
        ) {
          roles.add(binding.role);
        }
      }
    };
    collect(projectPolicy.bindings, true);
    collect(workspacePolicy?.bindings ?? [], !isSaaSMode);
    return [...roles];
  }, [identity, projectPolicy, workspacePolicy, isSaaSMode]);
}
