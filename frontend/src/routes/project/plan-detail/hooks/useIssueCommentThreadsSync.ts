import { useEffect } from "react";
import { useAppStore } from "@/stores/app";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";

// Comment writes never touch the issue's update time, so polling the issue
// cannot reveal another reviewer's reply, edit, resolve, or reopen. The
// threads are re-listed on their own cadence while the page is visible.
const THREADS_REFRESH_INTERVAL_MS = 15_000;

// Loads the issue's comments with thread replies when the issue changes
// server-side (polling bumps updateTime) or after local actions refresh it,
// and periodically in between for remote thread activity. Mounted once per
// plan page: both the Review Activity timeline and the statement editor read
// the same cache, so neither may depend on the other being expanded to have
// it filled.
export function useIssueCommentThreadsSync(issue: Issue | undefined) {
  const issueName = issue?.name;
  const updateKey = `${issue?.updateTime?.seconds ?? ""}:${issue?.updateTime?.nanos ?? ""}`;
  useEffect(() => {
    if (!issueName) return;
    const refresh = () => {
      void useAppStore
        .getState()
        .fetchIssueCommentThreads({ parent: issueName })
        .catch(() => undefined);
    };
    refresh();
    const timer = window.setInterval(() => {
      if (!document.hidden) refresh();
    }, THREADS_REFRESH_INTERVAL_MS);
    // Ticks are skipped while hidden; catch up as soon as the page is back.
    const onVisibilityChange = () => {
      if (!document.hidden) refresh();
    };
    document.addEventListener("visibilitychange", onVisibilityChange);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [issueName, updateKey]);
}
