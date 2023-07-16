<template>
  <NButton quaternary size="tiny" @click="state.showConfirmModal = true">
    <heroicons-outline:document-download class="w-5 h-auto" />
  </NButton>

  <BBModal
    v-if="state.showConfirmModal"
    :title="$t('common.export')"
    header-class="overflow-hidden"
    @close="state.showConfirmModal = false"
  >
    <div class="w-128 mb-6">
      {{ $t("issue.grant-request.data-export-attention") }}
    </div>
    <div class="w-full flex items-center justify-end mt-2 space-x-3 pr-1 pb-1">
      <button
        type="button"
        class="btn-cancel"
        @click="state.showConfirmModal = false"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        class="btn-primary"
        :disabled="state.isRequesting"
        @click="handleExportData"
      >
        <BBSpin v-if="state.isRequesting" class="mr-2" color="text-white" />
        {{ $t("common.confirm") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { ExportRecord } from "./types";
import { reactive } from "vue";
import {
  getExportFileType,
  getExportRequestFormat,
  pushNotification,
  useProjectIamPolicyStore,
  useSQLStore,
} from "@/store";
import dayjs from "dayjs";

interface LocalState {
  showConfirmModal: boolean;
  isRequesting: boolean;
}

const props = defineProps<{
  exportRecord: ExportRecord;
}>();

const sqlStore = useSQLStore();
const projectIamPolicyStore = useProjectIamPolicyStore();
const state = reactive<LocalState>({
  showConfirmModal: false,
  isRequesting: false,
});

const handleExportData = async () => {
  if (state.isRequesting) {
    return;
  }

  state.isRequesting = true;

  const exportRecord = props.exportRecord;
  const database = exportRecord.database;

  try {
    const { content } = await sqlStore.exportData({
      name: database.instance,
      connectionDatabase: database.databaseName,
      statement: exportRecord.statement,
      limit: exportRecord.maxRowCount,
      format: getExportRequestFormat(exportRecord.exportFormat),
      admin: false,
    });

    const blob = new Blob([content], {
      type: getExportFileType(exportRecord.exportFormat),
    });
    const url = window.URL.createObjectURL(blob);

    const fileFormat = exportRecord.exportFormat.toLowerCase();
    const formattedDateString = dayjs(new Date()).format("YYYY-MM-DDTHH-mm-ss");
    const filename = `export-data-${formattedDateString}`;
    const link = document.createElement("a");
    link.download = `${filename}.${fileFormat}`;
    link.href = url;
    link.click();
    // Fetch the latest iam policy.
    await projectIamPolicyStore.fetchProjectIamPolicy(database.project, true);
    state.isRequesting = false;
    state.showConfirmModal = false;
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: `Failed to export data`,
      description: JSON.stringify(error),
    });
  }
};
</script>
