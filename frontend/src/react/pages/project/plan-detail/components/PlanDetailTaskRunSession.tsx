import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { AlignHorizontalJustifyStart, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/react/components/ui/table";
import { useLivePoll } from "@/react/hooks/useLivePoll";
import { sameMessage } from "@/react/lib/protoIdentity";
import { getDateForPbTimestampProtoEs } from "@/types";
import {
  GetTaskRunSessionRequestSchema,
  type TaskRun,
  TaskRun_Status,
  type TaskRunSession_Postgres,
  type TaskRunSession_Postgres_Session,
  TaskRunSession_PostgresSchema,
} from "@/types/proto-es/v1/rollout_service_pb";

// A running task holds its session for the duration of the execution; refresh
// on this cadence so blocking/blocked sessions reflect the live database state
// rather than the moment the panel opened.
const SESSION_POLL_INTERVAL_MS = 5000;

export function PlanDetailTaskRunSession({
  taskRun,
  // When false, pause the live session poll — the card is mounted but its stage
  // is hidden. Defaults to live.
  active = true,
}: {
  taskRun: TaskRun;
  active?: boolean;
}) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [session, setSession] = useState<TaskRunSession_Postgres | undefined>();
  const isRunning = taskRun.status === TaskRun_Status.RUNNING;

  // Monotonic per-fetch sequence: two overlapping 5s poll ticks (or a tick still
  // in flight when the run settles) must not write a stale session over fresher
  // data. A response whose number is no longer the latest is dropped.
  const sessionFetchSeq = useRef(0);
  const fetchSession = useCallback(async () => {
    const seq = ++sessionFetchSeq.current;
    const response = await rolloutServiceClientConnect.getTaskRunSession(
      create(GetTaskRunSessionRequestSchema, { parent: taskRun.name }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
    if (seq !== sessionFetchSeq.current) {
      return;
    }
    const next =
      response.session.case === "postgres" ? response.session.value : undefined;
    // Keep the previous reference when the content is unchanged so the tables
    // don't re-render for nothing.
    setSession((prev) =>
      prev && next && sameMessage(TaskRunSession_PostgresSchema, prev, next)
        ? prev
        : next
    );
  }, [taskRun.name]);

  // Initial load (with spinner). Resetting session + loading when not running
  // keeps the panel consistent if the run finishes before the load resolves.
  useEffect(() => {
    if (!isRunning) {
      // Invalidate any in-flight fetch so it can't re-populate after the clear.
      sessionFetchSeq.current++;
      setSession(undefined);
      setLoading(false);
      return;
    }
    let canceled = false;
    setLoading(true);
    fetchSession()
      .catch(() => undefined)
      .finally(() => {
        if (!canceled) setLoading(false);
      });
    return () => {
      canceled = true;
    };
  }, [isRunning, fetchSession]);

  // Live refresh while running — swaps data in place (no spinner); a failing
  // tick is swallowed by useLivePoll.
  useLivePoll(active && isRunning, SESSION_POLL_INTERVAL_MS, fetchSession);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8 text-control-light">
        <Loader2 className="size-5 animate-spin" />
      </div>
    );
  }

  if (!session?.session) {
    return (
      <div className="rounded-md border bg-control-bg p-3 text-sm text-control-light">
        {t("task-run.no-session-found")}
      </div>
    );
  }

  return (
    <div className="flex w-full flex-col gap-y-2">
      <SessionTable rows={[session.session]} />
      {/* When nothing blocks (the common case), a one-line note replaces the
          header + description + empty table box. */}
      {session.blockingSessions.length > 0 ? (
        <div>
          <div className="flex items-center justify-start gap-x-1 pt-2">
            <span className="text-sm font-medium text-control">
              {t("task-run.blocking-sessions")}
            </span>
            <span className="textinfolabel">
              ({t("task-run.blocking-sessions-description")})
            </span>
          </div>
          <div className="mt-2">
            <SessionTable rows={session.blockingSessions} />
          </div>
        </div>
      ) : (
        <div className="text-sm text-control-light">
          {t("task-run.no-blocking-sessions")}
        </div>
      )}
      {session.blockedSessions.length > 0 && (
        <div>
          <div className="flex items-center justify-start gap-x-2 pt-2">
            <AlignHorizontalJustifyStart className="size-4 opacity-80" />
            <span className="text-sm font-medium text-control">
              {t("task-run.blocked-sessions")}
            </span>
            <span className="textinfolabel">
              {t("task-run.blocked-sessions-description")}
            </span>
          </div>
          <div className="mt-2">
            <SessionTable rows={session.blockedSessions} />
          </div>
        </div>
      )}
    </div>
  );
}

function SessionTable({ rows }: { rows: TaskRunSession_Postgres_Session[] }) {
  const normalizedRows = useMemo(() => rows ?? [], [rows]);

  if (normalizedRows.length === 0) {
    return (
      <div className="rounded-md border bg-control-bg p-3 text-sm text-control-light">
        -
      </div>
    );
  }

  return (
    <div className="overflow-auto rounded-sm border">
      <Table className="[&_td]:whitespace-nowrap [&_th]:whitespace-nowrap">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead>pid</TableHead>
            <TableHead>blocked_by_pids</TableHead>
            <TableHead>query</TableHead>
            <TableHead>state</TableHead>
            <TableHead>wait_event_type</TableHead>
            <TableHead>wait_event</TableHead>
            <TableHead>datname</TableHead>
            <TableHead>usename</TableHead>
            <TableHead>application_name</TableHead>
            <TableHead>client_addr</TableHead>
            <TableHead>client_port</TableHead>
            <TableHead>backend_start</TableHead>
            <TableHead>xact_start</TableHead>
            <TableHead>query_start</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {normalizedRows.map((row) => (
            <TableRow key={`${row.pid}-${row.queryStart?.seconds ?? 0}`}>
              <TableCell>{row.pid}</TableCell>
              <TableCell>{row.blockedByPids.join(", ") || "-"}</TableCell>
              <TableCell>
                <div className="max-w-[360px] truncate">{row.query || "-"}</div>
              </TableCell>
              <TableCell>{row.state || "-"}</TableCell>
              <TableCell>{row.waitEventType || "-"}</TableCell>
              <TableCell>{row.waitEvent || "-"}</TableCell>
              <TableCell>{row.datname || "-"}</TableCell>
              <TableCell>{row.usename || "-"}</TableCell>
              <TableCell>{row.applicationName || "-"}</TableCell>
              <TableCell>{row.clientAddr || "-"}</TableCell>
              <TableCell>{row.clientPort || "-"}</TableCell>
              <TableCell>{formatSessionTime(row.backendStart)}</TableCell>
              <TableCell>{formatSessionTime(row.xactStart)}</TableCell>
              <TableCell>{formatSessionTime(row.queryStart)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function formatSessionTime(
  ts: TaskRunSession_Postgres_Session["queryStart"] | undefined
) {
  const date = getDateForPbTimestampProtoEs(ts);
  if (!date) {
    return "-";
  }
  return date.toISOString().replace("T", " ").replace("Z", " UTC");
}
