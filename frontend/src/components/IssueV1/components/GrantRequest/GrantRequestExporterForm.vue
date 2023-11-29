<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start mb-4 space-y-4"
  >
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex items-center textlabel mb-2">
        {{ $t("common.database") }}
      </span>
      <div
        class="w-full flex flex-row justify-start items-start flex-wrap gap-2 gap-x-4"
      >
        <span
          v-if="
            exportMethod !== 'SQL' && selectedDatabaseResources.length === 0
          "
          >{{ $t("issue.grant-request.all-databases") }}</span
        >
        <DatabaseResourceTable
          v-else
          class="w-full"
          :database-resource-list="selectedDatabaseResources"
        />
      </div>
    </div>
    <div
      v-if="exportMethod === 'SQL'"
      class="w-full flex flex-col justify-start items-start"
    >
      <span class="flex items-center textlabel mb-2">SQL</span>
      <div class="w-full border rounded min-h-[6rem] relative">
        <MonacoEditor
          class="w-full h-[300px]"
          :content="state.statement"
          :auto-focus="false"
          :readonly="true"
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
import { MonacoEditor } from "@/components/MonacoEditor";
import { PresetRoleType } from "@/types";
import { DatabaseResource } from "@/types";
import { convertFromCELString } from "@/utils/issue/cel";
import { useIssueContext } from "../../logic";
import DatabaseResourceTable from "./DatabaseResourceTable.vue";

interface LocalState {
  maxRowCount: number;
  statement: string;
  expiredAt: string;
}

const { issue } = useIssueContext();
const state = reactive<LocalState>({
  maxRowCount: 1000,
  statement: "",
  expiredAt: "",
});
const selectedDatabaseResources = ref<DatabaseResource[]>([]);

const exportMethod = computed(() => {
  return state.statement === "" ? "DATABASE" : "SQL";
});

onMounted(async () => {
  const grantRequest = issue.value.grantRequest!;
  if (grantRequest.role !== PresetRoleType.EXPORTER) {
    throw "Only support EXPORTER role";
  }

  const conditionExpression = await convertFromCELString(
    grantRequest.condition?.expression ?? ""
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
});
</script>
