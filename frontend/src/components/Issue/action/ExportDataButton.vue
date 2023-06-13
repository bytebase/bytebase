<template>
  <button
    class="btn-primary flex flex-row justify-center items-center"
    @click="state.showConfirmModal = true"
  >
    <heroicons-outline:document-download class="w-4 h-auto mr-0.5" />
    {{ $t("common.export") }}
  </button>

  <BBModal
    v-if="state.showConfirmModal"
    :title="$t('common.export')"
    header-class="overflow-hidden"
    @confirm="handleExport"
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
        @click="handleExport"
      >
        <BBSpin v-if="state.isRequesting" class="mr-2" color="text-white" />
        {{ $t("common.confirm") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { head } from "lodash-es";
import { onMounted, reactive } from "vue";
import { useExtraIssueLogic, useIssueLogic } from "../logic";
import {
  GrantRequestPayload,
  Issue,
  PresetRoleType,
  UNKNOWN_ID,
} from "@/types";
import {
  getExportRequestFormat,
  pushNotification,
  useDatabaseV1Store,
  useProjectIamPolicyStore,
  useSQLStore,
} from "@/store";
import { BBSpin } from "@/bbkit";
import { convertFromCELString } from "@/utils/issue/cel";

interface LocalState {
  databaseId: string;
  statement: string;
  maxRowCount: number;
  exportFormat: "CSV" | "JSON";
  showConfirmModal: boolean;
  isRequesting: boolean;
}

const { changeIssueStatus } = useExtraIssueLogic();
const { issue } = useIssueLogic();
const databaseStore = useDatabaseV1Store();
const sqlStore = useSQLStore();
const projectIamPolicyStore = useProjectIamPolicyStore();
const state = reactive<LocalState>({
  databaseId: String(UNKNOWN_ID),
  statement: "",
  maxRowCount: 1000,
  exportFormat: "CSV",
  showConfirmModal: false,
  isRequesting: false,
});

onMounted(async () => {
  const payload = ((issue.value as Issue).payload as any)
    .grantRequest as GrantRequestPayload;
  if (payload.role !== PresetRoleType.EXPORTER) {
    throw "Only support EXPORTER role";
  }
  const conditionExpression = await convertFromCELString(
    payload.condition.expression
  );
  if (
    conditionExpression.databaseResources !== undefined &&
    conditionExpression.databaseResources.length > 0
  ) {
    const resource = head(conditionExpression.databaseResources);
    if (resource) {
      const database = await databaseStore.getOrFetchDatabaseByName(
        resource.databaseName
      );
      state.databaseId = database.uid;
    }
  }
  if (conditionExpression.statement !== undefined) {
    state.statement = conditionExpression.statement;
  }
  if (conditionExpression.rowLimit !== undefined) {
    state.maxRowCount = conditionExpression.rowLimit;
  }
  if (conditionExpression.exportFormat !== undefined) {
    state.exportFormat = conditionExpression.exportFormat as "CSV" | "JSON";
  }
});

const handleExport = async () => {
  if (state.isRequesting) {
    return;
  }
  state.isRequesting = true;

  const database = databaseStore.getDatabaseByUID(state.databaseId);
  if (database.uid === String(UNKNOWN_ID)) {
    throw "Database not found";
  }

  try {
    const { content } = await sqlStore.exportData({
      name: database.instance,
      connectionDatabase: database.databaseName,
      statement: state.statement,
      limit: state.maxRowCount,
      format: getExportRequestFormat(state.exportFormat),
    });
    const blob = new Blob([content], {
      type: state.exportFormat === "CSV" ? "text/csv" : "application/json",
    });
    const url = window.URL.createObjectURL(blob);
    const fileFormat = state.exportFormat.toLowerCase();
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
  // After data exported successfully, we change the issue status to DONE automatically.
  changeIssueStatus("DONE", "");
};
</script>
