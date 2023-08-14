<template>
  <div class="space-y-4">
    <div v-if="debugLogList.length > 0">
      <div class="flex flex-row justify-between items-center textinfolabel">
        {{
          $t("debug-log.count-of-logs", {
            count: debugLogList.length,
          })
        }}
        <button
          class="group btn-normal items-center !border-0 !bg-accent !text-white hover:!bg-indigo-500"
          @click="handleExport"
        >
          <heroicons-outline:document-arrow-down class="h-4 w-4 mr-1" />
          {{ $t("debug-log.table.operation.export") }}
        </button>
      </div>
      <DebugLogTable
        :debug-log-list="
          debugLogList.sort(
            (a, b) =>
              (b.recordTime?.getTime() ?? 0) - (a.recordTime?.getTime() ?? 0)
          )
        "
        @view-detail="
          (log: DebugLog) => {
            state.modalContent = log
            state.showModal = true;
            dialog!.open();
          }
        "
      />
      <BBDialog
        ref="dialog"
        :title="$t('debug-log.debug-log-detail')"
        :negative-text="$t('common.close')"
        :positive-text="$t('debug-log.table.operation.copy')"
        data-label="bb-migration-mode-dialog"
        :closable="true"
        @before-positive-click="handleCopy"
      >
        <div class="w-192 font-mono">
          <dl>
            <dd
              v-for="(value, key) in state.modalContent"
              :key="key"
              class="flex items-start text-sm md:mr-4 mb-1"
            >
              <NGrid x-gap="20" :cols="20">
                <NGi span="3">
                  <span class="textlabel whitespace-nowrap">{{
                    logKeyMap[key]
                  }}</span
                  ><span class="mr-1">:</span>
                </NGi>
                <NGi span="17">
                  <span v-if="value !== ''">
                    {{
                      key === "recordTime"
                        ? dayjs(value as Date).format("YYYY-MM-DD HH:mm:ss Z")
                        : value
                    }}
                  </span>
                  <span v-else class="italic text-gray-500">
                    {{ $t("debug-log.table.empty") }}
                  </span>
                </NGi>
              </NGrid>
            </dd>
          </dl>
        </div>
      </BBDialog>
    </div>
    <DebugLogEmptyView v-else />
  </div>
</template>

<script lang="ts" setup>
import { useClipboard } from "@vueuse/core";
import dayjs from "dayjs";
import download from "downloadjs";
import { NGrid, NGi } from "naive-ui";
import { reactive, ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { BBDialog } from "@/bbkit";
import { useDebugLogList, useNotificationStore } from "@/store";
import { DebugLog } from "@/types/proto/v1/actuator_service";

const dialog = ref<InstanceType<typeof BBDialog> | null>(null);

interface LocalState {
  showModal: boolean;
  modalContent?: DebugLog;
}

const state = reactive<LocalState>({
  showModal: false,
});

const logString = computed(() => {
  return JSON.stringify(state.modalContent);
});

const { t } = useI18n();
const { copy } = useClipboard({
  source: logString,
});
const notificationStore = useNotificationStore();
const debugLogList = useDebugLogList();

const logKeyMap = {
  recordTime: t("debug-log.table.record-ts"),
  method: t("debug-log.table.method"),
  requestPath: t("debug-log.table.request-path"),
  role: t("debug-log.table.role"),
  error: t("debug-log.table.error"),
  stackTrace: t("debug-log.table.stack-trace"),
};

const handleExport = () => {
  download(
    JSON.stringify(debugLogList.value, null, 2),
    `debuglog_${dayjs().format("YYYYMMDD")}.json`,
    "text/plain"
  );
};

const handleCopy = () => {
  copy();
  notificationStore.pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("debug-log.table.operation.copied"),
  });
  return true;
};
</script>
