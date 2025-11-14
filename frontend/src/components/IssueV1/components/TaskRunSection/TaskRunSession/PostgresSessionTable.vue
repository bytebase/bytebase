<template>
  <div class="w-full flex flex-col gap-y-2">
    <NDataTable
      :single-column="true"
      :columns="columns"
      :data="[session]"
      :row-key="rowKey"
      size="small"
    />

    <div class="pt-2 flex flex-row justify-start items-center">
      <span class="textlabel">
        {{ $t("issue.task-run.task-run-session.blocking-sessions.self") }}
      </span>
      <span class="textinfolabel ml-1">
        ({{
          $t("issue.task-run.task-run-session.blocking-sessions.description")
        }})
      </span>
    </div>
    <NDataTable
      size="small"
      :columns="columns"
      :data="blockingSessions"
      :row-key="rowKey"
    />

    <template v-if="blockedSessions.length > 0">
      <div class="pt-2 flex flex-row justify-start items-center">
        <AlignHorizontalJustifyStartIcon class="w-4 h-auto mr-2 opacity-80" />
        <p class="textlabel">
          {{ $t("issue.task-run.task-run-session.blocked-sessions.self") }}
        </p>
        <p class="textinfolabel ml-1">
          {{
            $t("issue.task-run.task-run-session.blocked-sessions.description")
          }}
        </p>
      </div>
      <NDataTable
        size="small"
        :columns="columns"
        :data="blockedSessions"
        :row-key="rowKey"
      />
    </template>
  </div>
</template>

<script setup lang="tsx">
import { AlignHorizontalJustifyStartIcon } from "lucide-vue-next";
import { type DataTableColumn, NDataTable } from "naive-ui";
import { computed } from "vue";
import { getDateForPbTimestampProtoEs } from "@/types";
import type {
  TaskRunSession_Postgres,
  TaskRunSession_Postgres_Session,
} from "@/types/proto-es/v1/rollout_service_pb";
import { StatementCell, TimeCell } from "./Cells";

const props = defineProps<{
  taskRunSession: TaskRunSession_Postgres;
}>();

const session = computed(() => props.taskRunSession.session!);
const blockingSessions = computed(() => props.taskRunSession.blockingSessions);
const blockedSessions = computed(() => props.taskRunSession.blockedSessions);

const rowKey = (session: TaskRunSession_Postgres_Session) => {
  return session.pid;
};

const columns = computed(() => {
  const columns: (DataTableColumn<TaskRunSession_Postgres_Session> & {
    hide?: boolean;
  })[] = [
    {
      key: "pid",
      title: () => "pid",
      width: 50,
      className: "whitespace-nowrap",
      render: (session) => {
        return session.pid;
      },
    },
    {
      key: "blocked_by_pids",
      title: () => "blocked_by_pids",
      width: 120,
      className: "whitespace-nowrap",
      render: (session) => {
        return session.blockedByPids.join(", ");
      },
    },
    {
      key: "query",
      resizable: true,
      title: () => "query",
      width: "60%",
      minWidth: 256,
      render: (session) => {
        return <StatementCell query={session.query} />;
      },
    },
    {
      key: "state",
      title: () => "state",
      width: 120,
      render: (session) => {
        return session.state;
      },
    },
    {
      key: "wait_event_type",
      title: () => "wait_event_type",
      width: 144,
      render: (session) => {
        return session.waitEventType;
      },
    },
    {
      key: "wait_event",
      title: () => "wait_event",
      width: 120,
      render: (session) => {
        return session.waitEvent;
      },
    },
    {
      key: "datname",
      title: () => "datname",
      width: 120,
      render: (session) => {
        return session.datname;
      },
    },
    {
      key: "usename",
      title: () => "usename",
      width: 120,
      render: (session) => {
        return session.usename;
      },
    },
    {
      key: "application_name",
      title: () => "application_name",
      width: 144,
      render: (session) => {
        return session.applicationName;
      },
    },
    {
      key: "client_addr",
      title: () => "client_addr",
      width: 120,
      render: (session) => {
        return session.clientAddr;
      },
    },
    {
      key: "client_port",
      width: 120,
      title: () => "client_port",
      render: (session) => {
        return session.clientPort;
      },
    },
    {
      key: "backend_start",
      title: () => "backend_start",
      width: 120,
      render: (session) => {
        return (
          <TimeCell date={getDateForPbTimestampProtoEs(session.backendStart)} />
        );
      },
    },
    {
      key: "xact_start",
      title: () => "xact_start",
      width: 120,
      render: (session) => {
        return (
          <TimeCell date={getDateForPbTimestampProtoEs(session.xactStart)} />
        );
      },
    },
    {
      key: "query_start",
      title: () => "query_start",
      width: 120,
      render: (session) => {
        return (
          <TimeCell date={getDateForPbTimestampProtoEs(session.queryStart)} />
        );
      },
    },
  ];
  return columns.filter((column) => !column.hide);
});
</script>
