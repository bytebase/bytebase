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
        <span v-if="selectedDatabaseResources.length === 0">{{
          $t("issue.grant-request.all-databases")
        }}</span>
        <DatabaseResourceTable
          v-else
          class="w-full"
          :database-resource-list="selectedDatabaseResources"
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
import { onMounted, reactive, ref } from "vue";
import type { DatabaseResource } from "@/types";
import { PresetRoleType } from "@/types";
import { convertFromCELString } from "@/utils/issue/cel";
import { useIssueContext } from "../../logic";
import DatabaseResourceTable from "./DatabaseResourceTable.vue";

interface LocalState {
  maxRowCount: number;
  expiredAt: string;
}

const { issue } = useIssueContext();
const state = reactive<LocalState>({
  maxRowCount: 1000,
  expiredAt: "",
});
const selectedDatabaseResources = ref<DatabaseResource[]>([]);

onMounted(async () => {
  const grantRequest = issue.value.grantRequest!;
  if (grantRequest.role !== PresetRoleType.PROJECT_EXPORTER) {
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
  if (conditionExpression.rowLimit !== undefined) {
    state.maxRowCount = conditionExpression.rowLimit;
  }
});
</script>
