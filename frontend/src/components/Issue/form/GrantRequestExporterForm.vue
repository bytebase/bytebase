<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start mb-4 space-y-4"
  >
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex items-center textlabel mb-2">
        {{ $t("common.database") }}
      </span>
      <DatabaseResourceTable
        class="w-full"
        :database-resource-list="selectedDatabaseResources"
      />
    </div>
    <div
      v-if="exportMethod === 'SQL'"
      class="w-full flex flex-col justify-start items-start"
    >
      <span class="flex items-center textlabel mb-2">SQL</span>
      <div class="w-full border rounded">
        <MonacoEditor
          class="w-full h-[300px] py-2"
          readonly
          :value="state.statement"
          :auto-focus="false"
          :language="'sql'"
        />
      </div>
    </div>
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex items-center textlabel mb-2">
        {{ $t("issue.grant-request.export-rows") }}
      </span>
      <div class="flex flex-row justify-start items-start">
        {{ state.maxRowCount }}
      </div>
    </div>
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex items-center textlabel mb-2">
        {{ $t("issue.grant-request.export-format") }}
      </span>
      <div class="flex flex-row justify-start items-start">
        {{ state.exportFormat }}
      </div>
    </div>
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex items-center textlabel mb-2">
        {{ $t("issue.grant-request.expired-at") }}
      </span>
      <div>
        {{ state.expiredAt }}
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { computed, onMounted, reactive, ref } from "vue";
import MonacoEditor from "@/components/MonacoEditor";
import { GrantRequestPayload, Issue, PresetRoleType } from "@/types";
import { DatabaseResource } from "@/types";
import { convertFromCELString } from "@/utils/issue/cel";
import { useIssueLogic } from "../logic";
import DatabaseResourceTable from "../table/DatabaseResourceTable.vue";

interface LocalState {
  maxRowCount: number;
  exportFormat: "CSV" | "JSON";
  statement: string;
  expiredAt: string;
}

const { issue } = useIssueLogic();
const state = reactive<LocalState>({
  maxRowCount: 1000,
  exportFormat: "CSV",
  statement: "",
  expiredAt: "",
});
const selectedDatabaseResources = ref<DatabaseResource[]>([]);

const exportMethod = computed(() => {
  return state.statement === "" ? "DATABASE" : "SQL";
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
    selectedDatabaseResources.value = conditionExpression.databaseResources;
  }
  if (conditionExpression.expiredTime !== undefined) {
    state.expiredAt = dayjs(new Date(conditionExpression.expiredTime)).format(
      "LLL"
    );
  } else {
    state.expiredAt = "-";
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
</script>
