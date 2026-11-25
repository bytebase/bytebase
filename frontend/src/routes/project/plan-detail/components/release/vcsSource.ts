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
  // The repository as its platform names it, such as "owner/name".
  repo: string;
  sha: string;
}

// Any client that can create a release writes its source URLs, so only a web
// URL becomes a link.
const webUrl = (url: string | undefined): string =>
  url && /^https?:\/\//i.test(url) ? url : "";

// The repository and commit of a release's source, from its commit URL. A
// release from an old action carries only the URL.
export const parseRepoCommit = (
  source: Release_VCSSource | undefined
): RepoCommit | undefined => {
  if (!source || !webUrl(source.url)) return undefined;
  const commitPath = COMMIT_PATH[source.vcsType];
  const index = commitPath ? source.url.lastIndexOf(commitPath) : -1;
  if (index < 0) return undefined;
  const repoUrl = source.url.slice(0, index);
  const sha = source.commitSha || source.url.slice(index + commitPath!.length);
  if (!repoUrl || !sha) return undefined;
  const repo = repoName(source.vcsType, repoUrl);
  if (!repo) return undefined;
  return { repoUrl, repo, sha };
};

// The URL's path after the host: "owner/name", or a GitLab subgroup's whole
// "group/subgroup/name". Azure DevOps puts "_git" in the path, and on
// dev.azure.com the organization too, so its name is "<project>/<repo>" from
// either side of "_git", or the repo alone when a pasted URL names no project.
const repoName = (vcsType: VCSType, repoUrl: string): string => {
  const host = /^https?:\/\/([^/]+)/i.exec(repoUrl)?.[1].toLowerCase() ?? "";
  const segments = repoUrl
    .replace(/^https?:\/\/[^/]+/i, "")
    .split("/")
    .filter(Boolean)
    .map(decodeSegment);
  if (vcsType === VCSType.AZURE_DEVOPS) {
    const git = segments.lastIndexOf("_git");
    if (git >= 0 && git < segments.length - 1) {
      const repo = segments[git + 1];
      const projectAt = host === "dev.azure.com" ? 1 : 0;
      return git > projectAt ? `${segments[git - 1]}/${repo}` : repo;
    }
  }
  return segments.join("/");
};

const decodeSegment = (segment: string): string => {
  try {
    return decodeURIComponent(segment);
  } catch {
    return segment;
  }
};

export const platformName = (vcsType: VCSType | undefined) =>
  vcsType === undefined ? undefined : PLATFORM_NAME[vcsType];

// A link to a release file at the release's commit. The path is the action's
// glob match, relative to the pipeline's working directory. A declarative
// rollout records its --file-pattern as the path, and a pattern with a glob
// metacharacter names no file; a versioned path is a real file name.
export const fileAtCommitUrl = (
  source: Release_VCSSource | undefined,
  path: string,
  declarative = false
): string | undefined => {
  const commit = parseRepoCommit(source);
  if (!commit || !source || !path) return undefined;
  if (declarative && /[*?[{]/.test(path)) return undefined;
  const { repoUrl, sha } = commit;
  // Each segment encoded, so "#", "?" and "&" in a file name stay in the path.
  const relative = path
    .replace(/^\.?\//, "")
    .split("/")
    .map(encodeURIComponent)
    .join("/");
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

// Commit and PR text is shown as plain text: trimmed, control and
// bidirectional-override characters removed, never rendered as markdown.
export const cleanSourceText = (text: string | undefined): string =>
  (text ?? "")
    .replace(/[\u0000-\u001f\u007f\u202a-\u202e\u2066-\u2069]/g, "")
    .trim();

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
  const href = webUrl(pr?.url || source.url);
  if (!href && !title && !author && !branch) return undefined;
  return {
    title: title || undefined,
    href,
    author: author || undefined,
    repoAtSha: commit ? `${commit.repo}@${commit.sha.slice(0, 7)}` : undefined,
    branch: branch || undefined,
  };
};
