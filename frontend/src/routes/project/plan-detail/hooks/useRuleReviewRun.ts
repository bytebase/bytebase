import { create } from "@bufbuild/protobuf";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { issueServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { useAppStore } from "@/stores/app";
import {
  GetReviewRunRequestSchema,
  IssueComment_ThreadState,
  type ReviewRun,
  ReviewRun_Status,
  ReviewRun_Type,
} from "@/types/proto-es/v1/issue_service_pb";

const POLL_INTERVAL_MS = 2000;
const EMPTY_ISSUE = "";

const isActive = (run: ReviewRun | null | undefined) =>
  run?.status === ReviewRun_Status.AVAILABLE ||
  run?.status === ReviewRun_Status.RUNNING;

export type RuleReviewRunState = {
  // undefined while loading; null when the review never ran on the issue.
  run: ReviewRun | null | undefined;
  loadFailed: boolean;
  openFindings: number;
  reload: () => Promise<void>;
};

// The issue's standard rule review run, read again whenever `reloadKey`
// changes and polled while it is queued or running. The server starts the
// review wherever it runs the plan checks, so the caller keys a reload on the
// same events.
export function useRuleReviewRun({
  enabled,
  issueName,
  reloadKey,
}: {
  enabled: boolean;
  issueName: string | undefined;
  reloadKey: string;
}): RuleReviewRunState {
  const reviewRunName =
    enabled && issueName ? `${issueName}/reviewRuns/rule` : undefined;
  const [run, setRun] = useState<ReviewRun | null | undefined>(undefined);
  const [loadFailed, setLoadFailed] = useState(false);

  const reload = useCallback(async () => {
    if (!reviewRunName) return;
    try {
      const resp = await issueServiceClientConnect.getReviewRun(
        create(GetReviewRunRequestSchema, { name: reviewRunName }),
        { contextValues: createContextValues().set(silentContextKey, true) }
      );
      setRun(resp);
      setLoadFailed(false);
    } catch (error) {
      if (ConnectError.from(error).code === Code.NotFound) {
        setRun(null);
        setLoadFailed(false);
      } else {
        setLoadFailed(true);
      }
    }
  }, [reviewRunName]);

  useEffect(() => {
    setRun(undefined);
    setLoadFailed(false);
  }, [reviewRunName]);

  useEffect(() => {
    void reload();
  }, [reload, reloadKey]);

  // Each poll stores a new run object, which schedules the next poll.
  useEffect(() => {
    if (!isActive(run)) return;
    const timer = window.setTimeout(() => void reload(), POLL_INTERVAL_MS);
    return () => window.clearTimeout(timer);
  }, [reload, run]);

  // A run posts its findings as comments when it completes; re-list them then
  // so they appear without waiting for the page's thread refresh.
  const active = isActive(run);
  const wasActive = useRef(false);
  useEffect(() => {
    if (wasActive.current && !active && run && issueName) {
      void useAppStore
        .getState()
        .fetchIssueCommentThreads({ parent: issueName })
        .catch(() => undefined);
    }
    wasActive.current = active;
  }, [active, issueName, run]);

  const comments = useAppStore((state) =>
    state.getIssueComments(issueName ?? EMPTY_ISSUE)
  );
  const openFindings = useMemo(
    () =>
      comments.filter(
        (comment) =>
          comment.reviewMetadata?.runType === ReviewRun_Type.RULE &&
          comment.threadState === IssueComment_ThreadState.OPEN
      ).length,
    [comments]
  );

  return useMemo(
    () => ({ run, loadFailed, openFindings, reload }),
    [loadFailed, openFindings, reload, run]
  );
}
