<template>
  <BBTable
    class="mt-2"
    :column-list="columnList"
    :data-source="auditLogList"
    :show-header="true"
    :row-clickable="false"
  >
    <template #body="{ rowData: auditLog }">
      <BBTableCell :left-padding="4" class="table-cell w-56">
        <div>
          {{ dayjs.unix(auditLog.createdTs).format("YYYY-MM-DD HH:mm:ss Z") }}
        </div>
      </BBTableCell>
      <BBTableCell class="table-cell w-20">
        {{ auditLog.creator }}
      </BBTableCell>
      <BBTableCell class="table-cell w-28">
        <span
          class="inline-flex items-center px-2 py-1 rounded-lg text-xs font-semibold whitespace-nowrap bg-green-100 text-green-800"
        >
          {{
            t(AuditActivityTypeI18nNameMap[auditLog.type as AuditActivityType])
          }}
        </span>
      </BBTableCell>
      <BBTableCell class="table-cell w-24">
        <div>{{ auditLog.level }}</div>
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
import {
  AuditLog,
  AuditActivityTypeI18nNameMap,
  AuditActivityType,
} from "@/types";

defineProps({
  auditLogList: {
    type: Array as PropType<AuditLog[]>,
    required: true,
  },
});

defineEmits<{
  (event: "view-detail", list: AuditLog, e: MouseEvent): void;
}>();

const { t } = useI18n();

const columnList = computed(() => [
  {
    title: t("audit-log.table.created-ts"),
  },
  {
    title: t("audit-log.table.creator"),
  },
  {
    title: t("audit-log.table.type"),
  },
  {
    title: t("audit-log.table.level"),
  },
  {
    title: t("audit-log.table.comment"),
  },
  {
    title: t("audit-log.table.payload"),
  },
]);
</script>
