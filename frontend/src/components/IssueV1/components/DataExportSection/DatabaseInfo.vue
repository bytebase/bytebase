<template>
  <div class="flex items-center flex-wrap gap-1">
    <InstanceV1Name
      :instance="coreDatabaseInfo.instanceEntity"
      :plain="true"
      :link="link"
    >
      <template
        v-if="
          database &&
          database.instanceEntity.environment !== database.effectiveEnvironment
        "
        #prefix
      >
        <EnvironmentV1Name
          :environment="database.instanceEntity.environmentEntity"
          :plain="true"
          :show-icon="false"
          :link="link"
          text-class="text-control-light"
        />
      </template>
    </InstanceV1Name>

    <heroicons-outline:chevron-right class="text-control-light" />

    <div class="flex items-center gap-x-1">
      <heroicons-outline:database />

      <EnvironmentV1Name
        :environment="coreDatabaseInfo.effectiveEnvironmentEntity"
        :plain="true"
        :show-icon="false"
        :link="link"
        text-class="text-control-light"
      />

      <DatabaseV1Name
        :database="coreDatabaseInfo"
        :plain="true"
        :link="link"
        :show-not-found="true"
      />

      <SQLEditorButtonV1 v-if="showSQLEditorButton" :database="database" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { SQLEditorButtonV1 } from "@/components/DatabaseDetail";
import { databaseForTask, useIssueContext } from "@/components/IssueV1/logic";
import { DatabaseV1Name, InstanceV1Name } from "@/components/v2";
import { usePageMode } from "@/store";
import { UNKNOWN_ID } from "@/types";

withDefaults(
  defineProps<{
    link?: boolean;
  }>(),
  {
    link: true,
  }
);

const { issue, selectedTask } = useIssueContext();
const pageMode = usePageMode();

const coreDatabaseInfo = computed(() => {
  return databaseForTask(issue.value, selectedTask.value);
});

const database = computedAsync(async () => {
  const maybeExistedDatabase = coreDatabaseInfo.value;
  if (maybeExistedDatabase.uid !== String(UNKNOWN_ID)) {
    return maybeExistedDatabase;
  }
  return undefined;
}, undefined);

const showSQLEditorButton = computed(() => {
  return pageMode.value === "BUNDLED" && database.value;
});
</script>
