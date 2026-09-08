import { useEffect } from "react";
import { useAppStore } from "@/stores/app";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";

// Loads the issue's comments with thread replies whenever the issue changes
// server-side (polling bumps updateTime) or after local actions refresh it.
// Mounted once per plan page: both the Review Activity timeline and the
// statement editor read the same cache, so neither may depend on the other
// being expanded to have it filled.
export function useIssueCommentThreadsSync(issue: Issue | undefined) {
  const issueName = issue?.name;
  const updateKey = `${issue?.updateTime?.seconds ?? ""}:${issue?.updateTime?.nanos ?? ""}`;
  useEffect(() => {
    if (!issueName) return;
    void useAppStore
      .getState()
      .fetchIssueCommentThreads({ parent: issueName })
      .catch(() => undefined);
  }, [issueName, updateKey]);
}
