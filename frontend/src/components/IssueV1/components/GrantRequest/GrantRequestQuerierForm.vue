<template>
  <div
    class="w-full mx-auto flex flex-col justify-start items-start gap-4 mb-4"
  >
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex w-full items-center textlabel mb-2">
        {{ $t("common.databases") }}
      </span>
      <div
        class="w-full flex flex-row justify-start items-start flex-wrap gap-2 gap-x-4"
      >
        <span v-if="state.selectedDatabaseResourceList.length === 0">{{
          $t("issue.grant-request.all-databases")
        }}</span>
        <DatabaseResourceTable
          v-else
          class="w-full"
          :database-resource-list="state.selectedDatabaseResourceList"
        />
      </div>
    </div>
    <div class="w-full flex flex-col justify-start items-start">
      <span class="flex w-full items-center textlabel mb-4">
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
import { onMounted, reactive } from "vue";
import { DatabaseResource, PresetRoleType } from "@/types";
import { convertFromCELString } from "@/utils/issue/cel";
import { useIssueContext } from "../../logic";
import DatabaseResourceTable from "./DatabaseResourceTable.vue";

interface LocalState {
  projectId?: string;
  allDatabases: boolean;
  selectedDatabaseResourceList: DatabaseResource[];
  expireDays: number;
  customDays: number;
  // For reviewing
  expiredAt: string;
}

const { issue } = useIssueContext();
const state = reactive<LocalState>({
  projectId: undefined,
  allDatabases: true,
  selectedDatabaseResourceList: [],
  expireDays: 7,
  customDays: 7,
  expiredAt: "",
});

onMounted(async () => {
  const grantRequest = issue.value.grantRequest!;
  if (grantRequest.role !== PresetRoleType.QUERIER) {
    throw "Only support QUERIER role";
  }

  const conditionExpression = await convertFromCELString(
    grantRequest.condition?.expression ?? ""
  );
  if (conditionExpression.expiredTime !== undefined) {
    state.expiredAt = dayjs(new Date(conditionExpression.expiredTime)).format(
      "LLL"
    );
  }
  if (
    conditionExpression.databaseResources !== undefined &&
    conditionExpression.databaseResources.length > 0
  ) {
    state.selectedDatabaseResourceList = conditionExpression.databaseResources;
  }
});
</script>
