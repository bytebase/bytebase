<template>
  <div class="w-full flex flex-col divide-y">
    <template v-if="stageList.length > 0">
      <div class="w-full flex flex-row items-center py-2 px-2 sm:px-4">
        <span class="textlabel mr-4">{{ $t("common.database") }}</span>
        <DatabaseInfo :database="database" />
      </div>
      <div class="w-full py-2 px-2 sm:px-4">
        <ExportOptionSection />
      </div>
    </template>
    <template v-else>
      <NoPermissionPlaceholder
        v-if="placeholder === 'PERMISSION_DENIED'"
        class="py-6"
      />
      <NEmpty v-else class="py-6" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { NEmpty } from "naive-ui";
import { computed, watch } from "vue";
import DatabaseInfo from "@/components/DatabaseInfo.vue";
import { useIssueContext } from "@/components/IssueV1/logic";
import { databaseForTask } from "@/components/Rollout/RolloutDetail";
import NoPermissionPlaceholder from "@/components/misc/NoPermissionPlaceholder.vue";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import { hasProjectPermissionV2 } from "@/utils";
import ExportOptionSection from "./ExportOptionSection";

const databaseStore = useDatabaseV1Store();
const { isCreating, issue, selectedTask } = useIssueContext();

// For database data export issue, the stageList should always be only 1 stage.
const stageList = computed(() => {
  return issue.value.rolloutEntity?.stages || [];
});

const database = computed(() => {
  return databaseForTask(issue.value.projectEntity, selectedTask.value);
});

const placeholder = computed(() => {
  if (
    isCreating.value &&
    !hasProjectPermissionV2(issue.value.projectEntity, "bb.rollouts.preview")
  ) {
    return "PERMISSION_DENIED";
  }
  return "NO_DATA";
});

// For data export issue, there should be only 1 database.
watch(
  () => database.value.name,
  async (databaseName) => {
    if (!databaseName || !isValidDatabaseName(databaseName)) {
      return;
    }
    await databaseStore.getOrFetchDatabaseByName(databaseName);
  },
  {
    immediate: true,
  }
);
</script>
