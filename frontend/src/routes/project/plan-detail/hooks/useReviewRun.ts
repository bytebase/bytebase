import { create } from "@bufbuild/protobuf";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { issueServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import { useAppStore } from "@/stores/app";
import {
  GetReviewRunRequestSchema,
  type ReviewRun,
  ReviewRun_State,
} from "@/types/proto-es/v1/issue_service_pb";
import { isReviewRunActive, reviewRunName } from "../utils/reviewRun";

const POLL_INTERVAL_MS = 2000;
const SETTLED_POLL_INTERVAL_MS = 15_000;

const sameTime = (a: Timestamp | undefined, b: Timestamp | undefined) =>
  a?.seconds === b?.seconds && a?.nanos === b?.nanos;

// Whether two reads show the same execution in the same state. A run restarts
// its create time on every re-run and stamps an end time on every completion.
const sameExecution = (a: ReviewRun | null | undefined, b: ReviewRun) =>
  !!a &&
  a.state === b.state &&
  sameTime(a.createTime, b.createTime) &&
  sameTime(a.endTime, b.endTime);

export type ReviewRunState = {
  // undefined while loading; null when the review never ran on the issue.
  run: ReviewRun | null | undefined;
  // True while `run` may not match the plan's SQL: the plan changed after it
  // was read, the server reports it outdated, or it shows a completion whose
  // findings are not yet listed.
  stale: boolean;
  loadFailed: boolean;
  reload: () => Promise<void>;
};

// The issue's review run, read again whenever `reloadKey` changes and polled:
// often while it is queued or running, and slowly once it has settled. The
// server starts the review when the plan's SQL changes, so the caller keys a
// reload on the plan's update.
export function useReviewRun({
  enabled,
  issueName,
  reloadKey,
}: {
  enabled: boolean;
  issueName: string | undefined;
  reloadKey: string;
}): ReviewRunState {
  const name = enabled && issueName ? reviewRunName(issueName) : undefined;
  const [run, setRun] = useState<ReviewRun | null | undefined>(undefined);
  const [readKey, setReadKey] = useState<string | undefined>(undefined);
  const [loadFailed, setLoadFailed] = useState(false);
  const [findingsPending, setFindingsPending] = useState(false);
  // Counts the reads that landed; each one schedules the next poll.
  const [reads, setReads] = useState(0);
  // Only the latest read lands: one it overtook would bring back an older run.
  const latestRead = useRef(0);
  const lastRun = useRef<ReviewRun | null | undefined>(undefined);

  const reload = useCallback(async () => {
    if (!name) return;
    const read = ++latestRead.current;
    try {
      const resp = await issueServiceClientConnect.getReviewRun(
        create(GetReviewRunRequestSchema, { name }),
        { contextValues: createContextValues().set(silentContextKey, true) }
      );
      if (read !== latestRead.current) return;
      // A run posts its findings when it succeeds, so a finished execution
      // this hook has not seen may come with comments the page has not
      // listed, even on the first read: the page's comment load may predate
      // the completion. Set with the run, so no render shows it completed
      // with old findings.
      if (
        resp.state === ReviewRun_State.SUCCEEDED &&
        !sameExecution(lastRun.current, resp)
      ) {
        setFindingsPending(true);
      }
      // A read that shows nothing new keeps the run it had, so a quiet poll
      // renders nothing.
      if (
        !sameExecution(lastRun.current, resp) ||
        lastRun.current?.outdated !== resp.outdated
      ) {
        lastRun.current = resp;
        setRun(resp);
      }
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
    setReads((count) => count + 1);
  }, [name, reloadKey]);

  useEffect(() => {
    latestRead.current++;
    lastRun.current = undefined;
    setRun(undefined);
    setReadKey(undefined);
    setLoadFailed(false);
    setFindingsPending(false);
    setReads(0);
  }, [name]);

  useEffect(() => {
    void reload();
  }, [reload]);

  // Every read schedules the next. An outdated run is polled like an active
  // one, until the server starts its replacement. A settled run is polled
  // too: someone else can rerun the review without changing the SQL, which
  // changes nothing the reload key holds.
  useEffect(() => {
    if (!name || reads === 0) return;
    const delay =
      isReviewRunActive(run) || run?.outdated
        ? POLL_INTERVAL_MS
        : SETTLED_POLL_INTERVAL_MS;
    const timer = window.setTimeout(() => void reload(), delay);
    return () => window.clearTimeout(timer);
  }, [name, reads, reload, run]);

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

  const stale = readKey !== reloadKey || findingsPending || !!run?.outdated;
  return useMemo(
    () => ({ run, stale, loadFailed, reload }),
    [loadFailed, reload, run, stale]
  );
}
