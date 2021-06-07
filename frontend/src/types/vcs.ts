export type VCSType = "GITLAB_SELF_HOST";

export interface VCSConfig {
  vcsType: VCSType;
  name: string;
  instanceURL: string;
  applicationId: string;
  secret: string;
}

export function isValidApplicationIdOrSecret(str: string): boolean {
  return /^[a-zA-Z0-9_]{64}$/.test(str);
}
