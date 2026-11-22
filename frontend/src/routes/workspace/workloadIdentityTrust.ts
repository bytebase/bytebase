import {
  isRepositoryProvider,
  parseSubjectPattern,
} from "@/components/WorkloadIdentityFlow/subjectPattern";
import {
  type WorkloadIdentityConfig,
  WorkloadIdentityConfig_ProviderType,
} from "@/types/proto-es/v1/workload_identity_service_pb";

// What a workload identity trusts, split the way the create form asks for it,
// so the list can say it in the form's words.
export interface WorkloadIdentityTrust {
  provider: "github" | "gitlab" | "oidc";
  primary:
    | { kind: "repository"; value: string }
    | { kind: "any-repository"; owner: string }
    | { kind: "pattern"; value: string }
    | { kind: "issuer"; value: string };
  secondary?:
    | { kind: "environment"; value: string }
    | { kind: "any-environment" }
    | { kind: "pattern"; value: string };
}

const issuerHost = (issuerUrl: string) => {
  try {
    return new URL(issuerUrl).host || issuerUrl;
  } catch {
    return issuerUrl;
  }
};

export const workloadIdentityTrust = (
  config: WorkloadIdentityConfig | undefined
): WorkloadIdentityTrust => {
  const providerType = config?.providerType;
  const pattern = config?.subjectPattern ?? "";
  if (providerType === undefined || !isRepositoryProvider(providerType)) {
    return {
      provider: "oidc",
      primary: { kind: "issuer", value: issuerHost(config?.issuerUrl ?? "") },
      secondary: pattern ? { kind: "pattern", value: pattern } : undefined,
    };
  }

  const provider =
    providerType === WorkloadIdentityConfig_ProviderType.GITHUB
      ? "github"
      : "gitlab";
  const parsed = parseSubjectPattern(providerType, pattern);
  if (!parsed) {
    return { provider, primary: { kind: "pattern", value: pattern } };
  }
  const { owner, repo } = parsed.fields;
  if (!repo) {
    return { provider, primary: { kind: "any-repository", owner } };
  }

  const primary = { kind: "repository" as const, value: `${owner}/${repo}` };
  if (parsed.rest === ":*") return { provider, primary };
  const environment = /^:environment:([^:]+)$/.exec(parsed.rest)?.[1];
  if (environment === "*") {
    return { provider, primary, secondary: { kind: "any-environment" } };
  }
  if (environment) {
    return {
      provider,
      primary,
      secondary: { kind: "environment", value: environment },
    };
  }
  return {
    provider,
    primary,
    secondary: { kind: "pattern", value: parsed.rest.slice(1) },
  };
};
