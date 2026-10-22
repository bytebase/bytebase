import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
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

const sameTime = (a: Timestamp | undefined, b: Timestamp | undefined) =>
  a?.seconds === b?.seconds && a?.nanos === b?.nanos;

// Whether two reads show the same execution in the same status. The slot
// restarts its create time on every reset and stamps an end time on every
// completion.
const sameExecution = (a: ReviewRun | null | undefined, b: ReviewRun) =>
  !!a &&
  a.status === b.status &&
  sameTime(a.createTime, b.createTime) &&
  sameTime(a.endTime, b.endTime);

export type RuleReviewRunState = {
  // undefined while loading; null when the review never ran on the issue.
  run: ReviewRun | null | undefined;
  // True while `run` may not match the server: the plan changed after it was
  // read, or it shows a completion whose findings are not yet listed.
  stale: boolean;
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
  const [readKey, setReadKey] = useState<string | undefined>(undefined);
  const [loadFailed, setLoadFailed] = useState(false);
  const [findingsPending, setFindingsPending] = useState(false);
  // Only the latest read lands: one it overtook would bring back an older run.
  const latestRead = useRef(0);
  const lastRun = useRef<ReviewRun | null | undefined>(undefined);

  const reload = useCallback(async () => {
    if (!reviewRunName) return;
    const read = ++latestRead.current;
    try {
      const resp = await issueServiceClientConnect.getReviewRun(
        create(GetReviewRunRequestSchema, { name: reviewRunName }),
        { contextValues: createContextValues().set(silentContextKey, true) }
      );
      if (read !== latestRead.current) return;
      // A run posts its findings with its DONE, so a DONE execution this hook
      // has not seen may come with comments the page has not listed, even on
      // the first read: the page's comment load may predate the completion.
      // Set with the run, so no render shows it completed with old findings.
      if (
        resp.status === ReviewRun_Status.DONE &&
        !sameExecution(lastRun.current, resp)
      ) {
        setFindingsPending(true);
      }
      lastRun.current = resp;
      setRun(resp);
      setReadKey(reloadKey);
      setLoadFailed(false);
    } catch (error) {
      if (read !== latestRead.current) return;
      if (ConnectError.from(error).code === Code.NotFound) {
        lastRun.current = null;
        setRun(null);
        setReadKey(reloadKey);
        setLoadFailed(false);
      } else {
        setLoadFailed(true);
      }
    }
  }, [reloadKey, reviewRunName]);

  useEffect(() => {
    latestRead.current++;
    lastRun.current = undefined;
    setRun(undefined);
    setReadKey(undefined);
    setLoadFailed(false);
    setFindingsPending(false);
  }, [reviewRunName]);

  useEffect(() => {
    void reload();
  }, [reload]);

  // Each poll stores a new run object, which schedules the next poll.
  useEffect(() => {
    if (!isActive(run)) return;
    const timer = window.setTimeout(() => void reload(), POLL_INTERVAL_MS);
    return () => window.clearTimeout(timer);
  }, [reload, run]);

  // A run posts its findings as comments when it completes; re-list them then
  // so they appear without waiting for the page's thread refresh. A failed
  // re-list leaves the findings pending.
  useEffect(() => {
    if (!findingsPending || !issueName) return;
    let current = true;
    void useAppStore
      .getState()
      .fetchIssueCommentThreads({ parent: issueName })
      .then(() => {
        if (current) setFindingsPending(false);
      })
      .catch(() => undefined);
    return () => {
      current = false;
    };
  }, [findingsPending, issueName]);

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

  const stale = readKey !== reloadKey || findingsPending;
  return useMemo(
    () => ({ run, stale, loadFailed, openFindings, reload }),
    [loadFailed, openFindings, reload, run, stale]
  );
}
