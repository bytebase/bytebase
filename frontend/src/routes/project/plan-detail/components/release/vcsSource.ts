import { VCSType } from "@/types/proto-es/v1/common_pb";
import type { Release_VCSSource } from "@/types/proto-es/v1/release_service_pb";

// The path each platform puts between the repository and the SHA in a commit
// URL, as bytebase-action builds it.
const COMMIT_PATH: Partial<Record<VCSType, string>> = {
  [VCSType.GITHUB]: "/commit/",
  [VCSType.GITLAB]: "/-/commit/",
  [VCSType.BITBUCKET]: "/commits/",
  [VCSType.AZURE_DEVOPS]: "/commit/",
};

const PLATFORM_NAME: Partial<Record<VCSType, string>> = {
  [VCSType.GITHUB]: "GitHub",
  [VCSType.GITLAB]: "GitLab",
  [VCSType.BITBUCKET]: "Bitbucket",
  [VCSType.AZURE_DEVOPS]: "Azure DevOps",
};

export interface RepoCommit {
  // The repository URL.
  repoUrl: string;
  // "owner/name": the repository URL's last two segments.
  repo: string;
  sha: string;
}

// The repository and commit of a release's source, from its commit URL. A
// release from an old action carries only the URL.
export const parseRepoCommit = (
  source: Release_VCSSource | undefined
): RepoCommit | undefined => {
  if (!source?.url) return undefined;
  const commitPath = COMMIT_PATH[source.vcsType];
  const index = commitPath ? source.url.lastIndexOf(commitPath) : -1;
  if (index < 0) return undefined;
  const repoUrl = source.url.slice(0, index);
  const sha = source.commitSha || source.url.slice(index + commitPath!.length);
  if (!repoUrl || !sha) return undefined;
  const segments = repoUrl.replace(/\/+$/, "").split("/");
  return { repoUrl, repo: segments.slice(-2).join("/"), sha };
};

export const platformName = (vcsType: VCSType | undefined) =>
  vcsType === undefined ? undefined : PLATFORM_NAME[vcsType];

// A link to a release file at the release's commit. The path is the action's
// glob match, relative to the pipeline's working directory.
export const fileAtCommitUrl = (
  source: Release_VCSSource | undefined,
  path: string
): string | undefined => {
  const commit = parseRepoCommit(source);
  if (!commit || !source || !path) return undefined;
  const { repoUrl, sha } = commit;
  const relative = path.replace(/^\.?\//, "");
  switch (source.vcsType) {
    case VCSType.GITHUB:
      return `${repoUrl}/blob/${sha}/${relative}`;
    case VCSType.GITLAB:
      return `${repoUrl}/-/blob/${sha}/${relative}`;
    case VCSType.BITBUCKET:
      return `${repoUrl}/src/${sha}/${relative}`;
    case VCSType.AZURE_DEVOPS:
      return `${repoUrl}?path=/${relative}&version=GC${sha}`;
    default:
      return undefined;
  }
};

export const SOURCE_TITLE_LIMIT = 72;

// Commit and PR text is shown as plain text: trimmed, control characters
// removed, never rendered as markdown.
export const cleanSourceText = (text: string | undefined): string =>
  (text ?? "").replace(/[\u0000-\u001f\u007f]/g, "").trim();

export interface SourceRow {
  // The PR title and number, else the commit title.
  title?: string;
  // The PR, else the commit; empty when the release carries no URL.
  href: string;
  author?: string;
  // "owner/name@sha7"; the link itself when there is no title.
  repoAtSha?: string;
  branch?: string;
}

// The details block's Source row (D9): title · author · repo@sha · branch,
// whatever subset the release carries. A CI the action does not detect passes
// only what its --source-* flags give.
export const buildSourceRow = (
  source: Release_VCSSource | undefined
): SourceRow | undefined => {
  if (!source) return undefined;
  const commit = parseRepoCommit(source);
  const pr =
    source.pullRequest && BigInt(source.pullRequest.number) > 0n
      ? source.pullRequest
      : undefined;
  let title = cleanSourceText(pr?.title || source.commitTitle);
  if (pr && title) {
    // A squash merge's commit title already ends with "(#N)".
    const marker = ` (#${pr.number})`;
    if (title.endsWith(marker)) title = title.slice(0, -marker.length);
    title = `${title} #${pr.number}`;
  }
  const author = cleanSourceText(source.authorName || source.authorLogin);
  const branch = cleanSourceText(source.branch);
  const href = pr?.url || source.url;
  if (!href && !title && !author && !branch) return undefined;
  return {
    title: title || undefined,
    href,
    author: author || undefined,
    repoAtSha: commit ? `${commit.repo}@${commit.sha.slice(0, 7)}` : undefined,
    branch: branch || undefined,
  };
};
