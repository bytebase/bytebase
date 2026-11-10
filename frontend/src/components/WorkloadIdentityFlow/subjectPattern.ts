import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/workload_identity_service_pb";

// The subject patterns the Repositories step reads and writes. The GitHub
// grammar is the one backend/plugin/idp/wif/github.go parses, so the form and
// the server split a pattern into the same parts:
//
//	repo:OWNER[@OWNER_ID]/*
//	repo:OWNER[@OWNER_ID]/REPO[@REPO_ID]:REST
//
// GitLab has no IDs in its subject:
//
//	project_path:GROUP/*
//	project_path:GROUP/PROJECT:REST

type RepositoryProvider =
  | WorkloadIdentityConfig_ProviderType.GITHUB
  | WorkloadIdentityConfig_ProviderType.GITLAB;

export const isRepositoryProvider = (
  providerType: WorkloadIdentityConfig_ProviderType
): providerType is RepositoryProvider =>
  providerType === WorkloadIdentityConfig_ProviderType.GITHUB ||
  providerType === WorkloadIdentityConfig_ProviderType.GITLAB;

export interface SubjectFields {
  owner: string;
  // Digits only, and always empty for GitLab.
  ownerId: string;
  // Empty when the pattern admits every repository of the owner.
  repo: string;
}

interface ParsedSubjectPattern {
  fields: SubjectFields;
  // Kept so that rewriting another field does not drop it; the form has no
  // field for it.
  repoId: string;
  // Everything after the repository, starting with ":", such as ":*" or
  // ":environment:prod". Empty when repo is empty.
  rest: string;
}

type SubjectFieldErrors = Partial<
  Record<keyof SubjectFields, "required" | "invalid">
>;

const ANY_REST = ":*";

const prefixOf = (provider: RepositoryProvider) =>
  provider === WorkloadIdentityConfig_ProviderType.GITHUB
    ? "repo:"
    : "project_path:";

const isDigits = (s: string) => /^\d+$/.test(s);

// A name segment holds no wildcard and none of the separators the pattern
// splits on. A GitLab project may name a subgroup, so it keeps "/".
const isValidName = (name: string, allowSlash = false) =>
  name !== "" && !(allowSlash ? /[\s*:@]/ : /[\s*:@/]/).test(name);

// Splits "name" or "name@123", as splitNameAndID does on the server.
const splitNameAndId = (
  segment: string
): { name: string; id: string } | undefined => {
  const at = segment.indexOf("@");
  const name = at < 0 ? segment : segment.slice(0, at);
  const id = at < 0 ? "" : segment.slice(at + 1);
  if (!isValidName(name)) return undefined;
  if (at >= 0 && !isDigits(id)) return undefined;
  return { name, id };
};

export const parseSubjectPattern = (
  provider: RepositoryProvider,
  pattern: string
): ParsedSubjectPattern | undefined => {
  const prefix = prefixOf(provider);
  if (!pattern.startsWith(prefix)) return undefined;
  const body = pattern.slice(prefix.length);
  const slash = body.indexOf("/");
  if (slash < 0) return undefined;
  const isGitHub = provider === WorkloadIdentityConfig_ProviderType.GITHUB;

  const owner = isGitHub
    ? splitNameAndId(body.slice(0, slash))
    : isValidName(body.slice(0, slash))
      ? { name: body.slice(0, slash), id: "" }
      : undefined;
  if (!owner) return undefined;

  const remainder = body.slice(slash + 1);
  if (remainder === "*") {
    return {
      fields: { owner: owner.name, ownerId: owner.id, repo: "" },
      repoId: "",
      rest: "",
    };
  }
  const colon = remainder.indexOf(":");
  if (colon < 0) return undefined;
  const repoSegment = remainder.slice(0, colon);
  const repo = isGitHub
    ? splitNameAndId(repoSegment)
    : isValidName(repoSegment, true)
      ? { name: repoSegment, id: "" }
      : undefined;
  if (!repo) return undefined;
  return {
    fields: { owner: owner.name, ownerId: owner.id, repo: repo.name },
    repoId: repo.id,
    rest: remainder.slice(colon),
  };
};

export const buildSubjectPattern = (
  provider: RepositoryProvider,
  fields: SubjectFields,
  repoId = "",
  rest = ANY_REST
): string => {
  const isGitHub = provider === WorkloadIdentityConfig_ProviderType.GITHUB;
  const owner =
    isGitHub && fields.ownerId
      ? `${fields.owner}@${fields.ownerId}`
      : fields.owner;
  if (!fields.repo) return `${prefixOf(provider)}${owner}/*`;
  const repo = isGitHub && repoId ? `${fields.repo}@${repoId}` : fields.repo;
  return `${prefixOf(provider)}${owner}/${repo}${rest}`;
};

export const validateSubjectFields = (
  provider: RepositoryProvider,
  fields: SubjectFields
): SubjectFieldErrors => {
  const isGitHub = provider === WorkloadIdentityConfig_ProviderType.GITHUB;
  const errors: SubjectFieldErrors = {};
  if (!fields.owner) errors.owner = "required";
  else if (!isValidName(fields.owner)) errors.owner = "invalid";
  if (isGitHub && fields.ownerId && !isDigits(fields.ownerId)) {
    errors.ownerId = "invalid";
  }
  if (fields.repo && !isValidName(fields.repo, !isGitHub)) {
    errors.repo = "invalid";
  }
  return errors;
};

type ReplaceResult =
  | { pattern: string }
  // Clearing the repository would drop a remainder such as
  // ":environment:prod", which widens the pattern.
  | { error: "repo-required" };

// Rewrites only the parts of a readable pattern that the fields name, so a
// remainder the fields cannot express survives a field edit.
export const replaceSubjectFields = (
  provider: RepositoryProvider,
  pattern: string,
  fields: SubjectFields
): ReplaceResult => {
  const parsed = parseSubjectPattern(provider, pattern);
  if (!parsed) return { pattern: buildSubjectPattern(provider, fields) };
  if (!fields.repo) {
    if (parsed.fields.repo && parsed.rest !== ANY_REST) {
      return { error: "repo-required" };
    }
    return { pattern: buildSubjectPattern(provider, fields) };
  }
  if (!parsed.fields.repo) {
    return { pattern: buildSubjectPattern(provider, fields) };
  }
  const repoId = parsed.fields.repo === fields.repo ? parsed.repoId : "";
  return {
    pattern: buildSubjectPattern(provider, fields, repoId, parsed.rest),
  };
};

// The environment a pattern pins jobs to, such as "prod" in
// "repo:o/r:environment:prod", or "*" for any environment.
export const subjectEnvironment = (pattern: string): string | undefined =>
  /:environment:([^:]+)/.exec(pattern)?.[1];
