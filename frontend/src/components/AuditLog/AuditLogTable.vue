<template>
  <BBTable
    class="mt-2"
    :column-list="columnList"
    :data-source="auditLogList"
    :show-header="true"
    :row-clickable="false"
  >
    <template #body="{ rowData: auditLog }: { rowData: LogEntity }">
      <BBTableCell :left-padding="4" class="table-cell w-56">
        <div>
          {{ dayjs(auditLog.createTime).format("YYYY-MM-DD HH:mm:ss Z") }}
        </div>
      </BBTableCell>
      <BBTableCell class="table-cell w-24">
        <span
          :class="`inline-flex items-center px-2 py-1 rounded-lg text-xs font-semibold whitespace-nowrap ${
            auditLevelColorMap[auditLog.level]
          }`"
          >{{ logEntity_LevelToJSON(auditLog.level) }}</span
        >
      </BBTableCell>
      <BBTableCell class="table-cell w-28 whitespace-nowrap">
        {{ t(AuditActivityTypeI18nNameMap[auditLog.action]) }}
      </BBTableCell>
      <BBTableCell class="table-cell w-20">
        <UserByEmail :email="auditLog.creator" :plain="true" />
      </BBTableCell>
      <BBTableCell class="table-cell w-36">
        <div v-if="auditLog.comment">
          {{ auditLog.comment }}
        </div>
        <span v-else class="italic text-gray-500">{{
          $t("audit-log.table.empty")
        }}</span>
      </BBTableCell>
      <BBTableCell class="table-cell w-28">
        <div>
          <div class="tooltip-wrapper">
            <div class="tooltip whitespace-nowrap">
              {{ $t("audit-log.table.view-details") }}
            </div>
            <button
              type="button"
              class="group btn-normal items-center !px-3 !text-accent hover:!bg-gray-50"
              @click.stop="
                () => {
                  $emit('view-detail', auditLog);
                }
              "
            >
              <heroicons-outline:document-magnifying-glass class="h-5 w-5" />
            </button>
          </div>
        </div>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { AuditActivityTypeI18nNameMap } from "@/types";
import {
  LogEntity,
  LogEntity_Level,
  logEntity_LevelToJSON,
} from "@/types/proto/v1/logging_service";
import { UserByEmail } from "../v2";

defineProps({
  auditLogList: {
    type: Array as PropType<LogEntity[]>,
    required: true,
  },
});

defineEmits<{
  (event: "view-detail", log: LogEntity, e: MouseEvent): void;
}>();

const { t } = useI18n();

const auditLevelColorMap: Record<LogEntity_Level, string> = {
  [LogEntity_Level.LEVEL_UNSPECIFIED]: "bg-gray-100 text-gray-800",
  [LogEntity_Level.UNRECOGNIZED]: "bg-gray-100 text-gray-800",
  [LogEntity_Level.LEVEL_INFO]: "bg-gray-100 text-gray-800",
  [LogEntity_Level.LEVEL_WARNING]: "bg-yellow-100 text-yellow-800",
  [LogEntity_Level.LEVEL_ERROR]: "bg-red-100 text-red-800",
};

const columnList = computed(() => [
  {
    title: t("audit-log.table.created-ts"),
  },
  {
    title: t("audit-log.table.level"),
  },
  {
    title: t("audit-log.table.type"),
  },
  {
    title: t("audit-log.table.actor"),
  },
  {
    title: t("audit-log.table.comment"),
  },
  {
    title: t("audit-log.table.payload"),
  },
]);
</script>
