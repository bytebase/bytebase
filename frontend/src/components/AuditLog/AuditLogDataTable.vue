<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="auditLogList"
    :striped="true"
    :loading="loading"
    :row-key="(data: AuditLog) => data.name"
    @update:sorter="$emit('update:sorters', $event)"
  />
</template>

<script lang="tsx" setup>
import { file_google_rpc_error_details } from "@buf/googleapis_googleapis.bufbuild_es/google/rpc/error_details_pb";
import { createRegistry, toJsonString } from "@bufbuild/protobuf";
import { AnySchema } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { ExternalLinkIcon } from "lucide-vue-next";
import {
  type DataTableColumn,
  type DataTableSortState,
  NButton,
  NDataTable,
} from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { ProjectV1Name } from "@/components/v2";
import { UserLink } from "@/components/v2/Model/cells";
import { mapSorterStatus } from "@/components/v2/Model/utils";
import {
  extractUserId,
  getProjectIdPlanUidStageUidFromRolloutName,
  projectNamePrefix,
  rolloutNamePrefix,
} from "@/store/modules/v1/common";
import { emptyProject, getDateForPbTimestampProtoEs } from "@/types";
import { StatusSchema } from "@/types/proto-es/google/rpc/status_pb";
import type { AuditLog } from "@/types/proto-es/v1/audit_log_service_pb";
import {
  AuditDataSchema,
  AuditLog_Severity,
} from "@/types/proto-es/v1/audit_log_service_pb";
import { IssueService } from "@/types/proto-es/v1/issue_service_pb";
import {
  file_v1_plan_service,
  PlanService,
} from "@/types/proto-es/v1/plan_service_pb";
import { RolloutService } from "@/types/proto-es/v1/rollout_service_pb";
import { SettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import { SQLService } from "@/types/proto-es/v1/sql_service_pb";
import { extractProjectResourceName, humanizeDurationV1 } from "@/utils";
import JSONStringView from "./JSONStringView.vue";

type AuditDataTableColumn = DataTableColumn<AuditLog> & {
  hide?: boolean;
};

// The registry is used to decode anypb protobuf messages to JSON.
const registry = createRegistry(
  file_google_rpc_error_details,
  file_v1_plan_service,
  AuditDataSchema,
  SettingSchema
);

const props = withDefaults(
  defineProps<{
    auditLogList: AuditLog[];
    showProject?: boolean;
    loading?: boolean;
    sorters?: DataTableSortState[];
  }>(),
  {
    showProject: true,
    loading: false,
  }
);

defineEmits<{
  (event: "update:sorters", sorters: DataTableSortState[]): void;
}>();

const { t } = useI18n();

const columnList = computed((): AuditDataTableColumn[] => {
  const columns: AuditDataTableColumn[] = (
    [
      {
        key: "create_time",
        title: t("audit-log.table.created-ts"),
        width: 220,
        resizable: true,
        render: (auditLog) =>
          dayjs(getDateForPbTimestampProtoEs(auditLog.createTime)).format(
            "YYYY-MM-DD HH:mm:ss Z"
          ),
      },
      {
        key: "severity",
        width: 60,
        title: t("audit-log.table.level"),
        render: (auditLog) => AuditLog_Severity[auditLog.severity],
      },
      {
        key: "project",
        width: 120,
        title: t("common.project"),
        resizable: true,
        hide: !props.showProject,
        render: (auditLog) => {
          const projectResourceId = extractProjectResourceName(auditLog.name);
          if (!projectResourceId) {
            return <span>-</span>;
          }
          const mockProject = {
            ...emptyProject(),
            name: `${projectNamePrefix}${projectResourceId}`,
            title: projectResourceId,
          };
          return <ProjectV1Name project={mockProject} />;
        },
      },
      {
        key: "method",
        resizable: true,
        width: 256,
        title: t("audit-log.table.method"),
        render: (auditLog) => auditLog.method,
      },
      {
        key: "actor",
        width: 180,
        title: t("audit-log.table.actor"),
        resizable: true,
        render: (auditLog) => {
          if (!auditLog.user) {
            return <span>-</span>;
          }
          const email = extractUserId(auditLog.user);
          return <UserLink title={email} email={email} />;
        },
      },
      {
        key: "request",
        resizable: true,
        minWidth: 256,
        width: 256,
        title: t("audit-log.table.request"),
        render: (auditLog) =>
          auditLog.request.length > 0 ? (
            <JSONStringView jsonString={auditLog.request} />
          ) : (
            "-"
          ),
      },
      {
        key: "response",
        resizable: true,
        minWidth: 256,
        width: 256,
        title: t("audit-log.table.response"),
        render: (auditLog) =>
          auditLog.response.length > 0 ? (
            <JSONStringView jsonString={auditLog.response} />
          ) : (
            "-"
          ),
      },
      {
        key: "status",
        resizable: true,
        width: 96,
        title: t("audit-log.table.status"),
        render: (auditLog) =>
          auditLog.status ? (
            <JSONStringView
              jsonString={toJsonString(StatusSchema, auditLog.status, {
                registry: registry,
              })}
            />
          ) : (
            "-"
          ),
      },
      {
        key: "latency",
        width: 96,
        title: t("audit-log.table.latency"),
        render: (auditLog) => {
          return <span>{humanizeDurationV1(auditLog.latency)}</span>;
        },
      },
      {
        key: "service-data",
        resizable: true,
        minWidth: 256,
        width: 256,
        title: t("audit-log.table.service-data"),
        render: (auditLog) => {
          return auditLog.serviceData ? (
            <JSONStringView
              jsonString={toJsonString(AnySchema, auditLog.serviceData, {
                registry: registry,
              })}
            />
          ) : (
            "-"
          );
        },
      },
      {
        key: "view",
        width: 60,
        title: t("common.view"),
        render: (auditLog) => {
          let link = getViewLink(auditLog);
          if (!link) {
            return null;
          }
          if (!link.startsWith("/")) {
            link = `/${link}`;
          }
          return (
            <a href={link} target="_blank">
              <NButton size="small" text type="primary">
                <ExternalLinkIcon class={"w-4"} />
              </NButton>
            </a>
          );
        },
      },
    ] as AuditDataTableColumn[]
  ).filter((column) => !column.hide);
  return mapSorterStatus(columns, props.sorters);
});

const getViewLink = (auditLog: AuditLog): string | null => {
  let parsedRequest: Record<string, unknown>;
  let parsedResponse: Record<string, unknown>;
  try {
    parsedRequest = JSON.parse(auditLog.request || "{}") as Record<
      string,
      unknown
    >;
    parsedResponse = JSON.parse(auditLog.response || "{}") as Record<
      string,
      unknown
    >;
  } catch {
    return null;
  }
  if (Boolean(parsedRequest["validateOnly"])) {
    return null;
  }

  const sections = auditLog.method.split("/").filter((i) => i);
  switch (sections[0]) {
    case RolloutService.typeName:
      return (parsedResponse["name"] as string) || null;
    case PlanService.typeName:
      return (parsedResponse["name"] as string) || null;
    case IssueService.typeName:
      return (parsedResponse["name"] as string) || null;
    case SQLService.typeName: {
      if (sections[1] !== "Export") {
        return null;
      }
      const name = parsedRequest["name"] as string | undefined;
      if (!name) {
        return null;
      }
      const [projectId, planId, _] =
        getProjectIdPlanUidStageUidFromRolloutName(name);
      if (!projectId || !planId) {
        return null;
      }
      return `${projectNamePrefix}${projectId}/${rolloutNamePrefix}${planId}`;
    }
  }
  return null;
};
</script>
