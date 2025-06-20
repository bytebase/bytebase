<template>
  <NDataTable
    size="small"
    :columns="columnList"
    :data="auditLogList"
    :striped="true"
    :loading="loading"
    :row-key="(data: AuditLog) => data.name"
  />
</template>

<script lang="tsx" setup>
import { fromBinary } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { NDataTable, type DataTableColumn } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import BBAvatar from "@/bbkit/BBAvatar.vue";
import { useProjectV1Store, useUserStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { getDateForPbTimestampProtoEs } from "@/types";
import type { AuditLog } from "@/types/proto-es/v1/audit_log_service_pb";
import { AuditDataSchema } from "@/types/proto-es/v1/audit_log_service_pb";
import { SettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import { extractProjectResourceName } from "@/utils";
import JSONStringView from "./JSONStringView.vue";

type AuditDataTableColumn = DataTableColumn<AuditLog> & {
  hide?: boolean;
};

const props = withDefaults(
  defineProps<{
    auditLogList: AuditLog[];
    showProject?: boolean;
    loading?: boolean;
  }>(),
  {
    showProject: true,
    loading: false,
  }
);

const { t } = useI18n();
const projectStore = useProjectV1Store();
const userStore = useUserStore();

const columnList = computed((): AuditDataTableColumn[] => {
  return (
    [
      {
        key: "created-ts",
        title: t("audit-log.table.created-ts"),
        width: 240,
        render: (auditLog) =>
          dayjs(getDateForPbTimestampProtoEs(auditLog.createTime)).format(
            "YYYY-MM-DD HH:mm:ss Z"
          ),
      },
      {
        key: "severity",
        width: 96,
        title: t("audit-log.table.level"),
        render: (auditLog) => auditLog.severity,
      },
      {
        key: "project",
        width: 96,
        title: t("common.project"),
        hide: !props.showProject,
        render: (auditLog) => {
          const projectResourceId = extractProjectResourceName(auditLog.name);
          if (!projectResourceId) {
            return <span>-</span>;
          }
          const project = projectStore.getProjectByName(
            `${projectNamePrefix}${projectResourceId}`
          );
          return <span>{project.title}</span>;
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
        width: 128,
        title: t("audit-log.table.actor"),
        render: (auditLog) => {
          const user = userStore.getUserByIdentifier(auditLog.user);
          if (!user) {
            return <span>-</span>;
          }
          return (
            <div class="flex flex-row items-center overflow-hidden gap-x-1">
              <BBAvatar size="SMALL" username={user.title} />
              <span class="truncate">{user.title}</span>
            </div>
          );
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
        minWidth: 256,
        width: 256,
        title: t("audit-log.table.status"),
        render: (auditLog) =>
          auditLog.status ? (
            <JSONStringView jsonString={JSON.stringify(auditLog.status)} />
          ) : (
            "-"
          ),
      },
      {
        key: "service-data",
        resizable: true,
        minWidth: 256,
        width: 256,
        title: t("audit-log.table.service-data"),
        render: (auditLog) => {
          return auditLog.serviceData && auditLog.serviceData.typeUrl ? (
            <JSONStringView
              jsonString={JSON.stringify(
                {
                  "@type": auditLog.serviceData.typeUrl,
                  ...getServiceDataValue(
                    auditLog.serviceData.typeUrl,
                    auditLog.serviceData.value
                  ),
                },
                (_, value) => {
                  if (typeof value === "bigint") {
                    return value.toString(); // Convert to string
                  }
                  return value;
                }
              )}
            />
          ) : (
            "-"
          );
        },
      },
    ] as AuditDataTableColumn[]
  ).filter((column) => !column.hide);
});

function getServiceDataValue(typeUrl: string, value: Uint8Array): any {
  switch (typeUrl) {
    case "type.googleapis.com/bytebase.v1.AuditData":
      return fromBinary(AuditDataSchema, value);
    case "type.googleapis.com/bytebase.v1.Setting":
      return fromBinary(SettingSchema, value);
    default:
      return null;
  }
}
</script>
