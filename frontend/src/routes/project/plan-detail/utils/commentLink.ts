import { issueCommentNamePrefix } from "@/stores/modules/v1/common";

// A comment's address is the plan's URL plus the last two segments of the
// comment's name as the fragment: `#issueComments/{id}`. The plan names its
// issue, so the fragment and the issue name make the comment's name again.

export function commentFragmentOf(commentName: string): string | undefined {
  const index = commentName.lastIndexOf(`/${issueCommentNamePrefix}`);
  if (index < 0) return undefined;
  const fragment = commentName.slice(index + 1);
  return fragment.length > issueCommentNamePrefix.length ? fragment : undefined;
}

export function commentNameOfHash(
  hash: string | undefined,
  issueName: string | undefined
): string | undefined {
  if (!hash || !issueName) return undefined;
  let fragment: string;
  try {
    fragment = decodeURIComponent(hash.replace(/^#/, ""));
  } catch {
    return undefined;
  }
  if (!fragment.startsWith(issueCommentNamePrefix)) return undefined;
  const id = fragment.slice(issueCommentNamePrefix.length);
  if (!id || id.includes("/")) return undefined;
  return `${issueName}/${fragment}`;
}
