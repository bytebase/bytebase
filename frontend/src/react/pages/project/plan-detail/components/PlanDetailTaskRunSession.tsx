import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { AlignHorizontalJustifyStart, Loader2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
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
import { getDateForPbTimestampProtoEs } from "@/types";
import {
  GetTaskRunSessionRequestSchema,
  type TaskRun,
  TaskRun_Status,
  type TaskRunSession_Postgres,
  type TaskRunSession_Postgres_Session,
} from "@/types/proto-es/v1/rollout_service_pb";

export function PlanDetailTaskRunSession({ taskRun }: { taskRun: TaskRun }) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [session, setSession] = useState<TaskRunSession_Postgres | undefined>();

  useEffect(() => {
    let canceled = false;

    const load = async () => {
      if (taskRun.status !== TaskRun_Status.RUNNING) {
        setSession(undefined);
        return;
      }

      try {
        setLoading(true);
        const response = await rolloutServiceClientConnect.getTaskRunSession(
          create(GetTaskRunSessionRequestSchema, {
            parent: taskRun.name,
          }),
          {
            contextValues: createContextValues().set(silentContextKey, true),
          }
        );
        if (canceled) {
          return;
        }
        if (response.session.case === "postgres") {
          setSession(response.session.value);
        } else {
          setSession(undefined);
        }
      } finally {
        if (!canceled) {
          setLoading(false);
        }
      }
    };

    void load();
    return () => {
      canceled = true;
    };
  }, [taskRun.name, taskRun.status]);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-8 text-control-light">
        <Loader2 className="h-5 w-5 animate-spin" />
      </div>
    );
  }

  if (!session?.session) {
    return (
      <div className="rounded-md border bg-gray-50 p-3 text-sm text-control-light">
        {t("task-run.no-session-found")}
      </div>
    );
  }

  return (
    <div className="flex w-full flex-col gap-y-2">
      <SessionTable rows={[session.session]} />
      <div>
        <div className="flex items-center justify-start gap-x-1 pt-2">
          <span className="textlabel">{t("task-run.blocking-sessions")}</span>
          <span className="textinfolabel">
            ({t("task-run.blocking-sessions-description")})
          </span>
        </div>
        <div className="mt-2">
          <SessionTable rows={session.blockingSessions} />
        </div>
      </div>
      {session.blockedSessions.length > 0 && (
        <div>
          <div className="flex items-center justify-start gap-x-2 pt-2">
            <AlignHorizontalJustifyStart className="h-4 w-4 opacity-80" />
            <span className="textlabel">{t("task-run.blocked-sessions")}</span>
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
      <div className="rounded-md border bg-gray-50 p-3 text-sm text-control-light">
        -
      </div>
    );
  }

  return (
    <div className="overflow-auto rounded-sm border">
      <Table className="table-fixed">
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="w-20">pid</TableHead>
            <TableHead className="w-28">blocked_by_pids</TableHead>
            <TableHead className="min-w-[256px]">query</TableHead>
            <TableHead className="w-28">state</TableHead>
            <TableHead className="w-28">wait_event_type</TableHead>
            <TableHead className="w-28">wait_event</TableHead>
            <TableHead className="w-28">datname</TableHead>
            <TableHead className="w-28">usename</TableHead>
            <TableHead className="w-36">application_name</TableHead>
            <TableHead className="w-28">client_addr</TableHead>
            <TableHead className="w-24">client_port</TableHead>
            <TableHead className="w-40">backend_start</TableHead>
            <TableHead className="w-40">xact_start</TableHead>
            <TableHead className="w-40">query_start</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {normalizedRows.map((row) => (
            <TableRow key={`${row.pid}-${row.queryStart?.seconds ?? 0}`}>
              <TableCell>{row.pid}</TableCell>
              <TableCell>{row.blockedByPids.join(", ") || "-"}</TableCell>
              <TableCell className="truncate">{row.query || "-"}</TableCell>
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
