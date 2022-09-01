<template>
  <BBTable
    class="mt-2"
    :column-list="columnList"
    :data-source="debugLogList"
    :show-header="true"
    :row-clickable="false"
  >
    <template #body="{ rowData: debugLog }">
      <BBTableCell :left-padding="4" class="table-cell w-56">
        <div>
          {{ dayjs.unix(debugLog.RecordTs).format("YYYY-MM-DD HH:mm:ss Z") }}
        </div>
      </BBTableCell>
      <BBTableCell class="table-cell w-20">
        <HTTPMethodTag :method="debugLog.Method" />
      </BBTableCell>
      <BBTableCell class="table-cell w-28">
        <div>{{ debugLog.RequestPath }}</div>
      </BBTableCell>
      <BBTableCell class="table-cell w-20">
        <span v-if="!debugLog.Role" class="italic text-gray-500">{{
          $t("debug-log.table.empty")
        }}</span>
        <div v-else>{{ debugLog.Role }}</div>
      </BBTableCell>
      <BBTableCell class="table-cell pt-3">
        <EllipsisText :line-clamp="2">{{ debugLog.Error }}</EllipsisText>
      </BBTableCell>
      <BBTableCell class="table-cell w-28">
        <div class="tooltip-wrapper">
          <div class="tooltip whitespace-nowrap">
            {{ $t("debug-log.table.operation.view-details") }}
          </div>
          <button
            type="button"
            class="group btn-normal items-center !px-3 !text-accent hover:!bg-gray-50"
            @click.stop="
              () => {
                $emit('view-detail', debugLog);
              }
            "
          >
            <heroicons-outline:document-magnifying-glass class="h-5 w-5" />
          </button>
        </div>
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { useI18n } from "vue-i18n";
import { DebugLog } from "@/types/debug";
import HTTPMethodTag from "./HTTPMethodTag.vue";
import EllipsisText from "../EllipsisText.vue";

defineProps({
  debugLogList: {
    type: Array as PropType<DebugLog[]>,
    required: true,
  },
});

defineEmits<{
  (event: "view-detail", list: DebugLog, e: MouseEvent): void;
}>();

const { t } = useI18n();

const columnList = computed(() => [
  {
    title: t("debug-log.table.record-ts"),
  },
  {
    title: t("debug-log.table.method"),
  },
  {
    title: t("debug-log.table.request-path"),
  },
  {
    title: t("debug-log.table.role"),
  },
  {
    title: t("debug-log.table.error"),
  },
  {
    title: t("debug-log.table.operation.operation"),
  },
]);
</script>
