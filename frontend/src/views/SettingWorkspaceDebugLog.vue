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
        :debug-log-list="debugLogList"
        @view-detail="
          (log: any) => {
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
                      key === "RecordTs"
                        ? dayjs.unix(value as number).format("YYYY-MM-DD HH:mm:ss Z")
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
    <DebugLogsEmptyView v-else />
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref, computed } from "vue";
import { NGrid, NGi } from "naive-ui";
import { useI18n } from "vue-i18n";
import download from "downloadjs";
import dayjs from "dayjs";
import { useClipboard } from "@vueuse/core";
import { BBDialog } from "@/bbkit";
import { useDebugLogList, useNotificationStore } from "@/store";

const dialog = ref<InstanceType<typeof BBDialog> | null>(null);
const state = reactive({
  showModal: false,
  modalContent: {},
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
  RecordTs: t("debug-log.table.record-ts"),
  Method: t("debug-log.table.method"),
  RequestPath: t("debug-log.table.request-path"),
  Role: t("debug-log.table.role"),
  Error: t("debug-log.table.error"),
  StackTrace: t("debug-log.table.stack-trace"),
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
