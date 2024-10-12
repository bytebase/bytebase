<template>
  <SyncDatabaseSchema
    :key="project.name"
    :project="project"
    :source="source"
    :source-schema-type="source ? 'SCHEMA_HISTORY_VERSION' : undefined"
    :target-database-list="targetDatabaseList"
  />
</template>

<script setup lang="ts">
import { computed, watchEffect, ref, onMounted } from "vue";
import { useRoute } from "vue-router";
import SyncDatabaseSchema from "@/components/SyncDatabaseSchema/index.vue";
import { type ChangeHistorySourceSchema } from "@/components/SyncDatabaseSchema/types";
import {
  useProjectByName,
  useDatabaseV1Store,
  useChangeHistoryStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { UNKNOWN_DATABASE_NAME } from "@/types";
import { ChangeHistoryView } from "@/types/proto/v1/database_service";
import { extractDatabaseNameAndChangeHistoryUID } from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const route = useRoute();
const changeHistoryStore = useChangeHistoryStore();
const databaseStore = useDatabaseV1Store();
const targetDatabaseList = ref<string[]>([]);

const changeHistoryVersion = computed(() => {
  const version = route.query.version as string;
  return version || "";
});

const changeHistory = computed(() => {
  if (!changeHistoryVersion.value) {
    return;
  }
  return changeHistoryStore.getChangeHistoryByName(changeHistoryVersion.value);
});

onMounted(async () => {
  if (!changeHistoryVersion.value) {
    return;
  }
  const history = await changeHistoryStore.getOrFetchChangeHistoryByName(
    changeHistoryVersion.value,
    ChangeHistoryView.CHANGE_HISTORY_VIEW_FULL
  );
  if (!history) {
    return;
  }
  const { databaseName } = extractDatabaseNameAndChangeHistoryUID(history.name);
  await databaseStore.getOrFetchDatabaseByName(databaseName);

  const target = route.query.target as string;
  const databaseList = (target || "").split(",").filter((n) => n);

  const databaseNameList = [];
  for (const dbName of databaseList) {
    const database = await databaseStore.getOrFetchDatabaseByName(dbName);
    if (database.name !== UNKNOWN_DATABASE_NAME) {
      databaseNameList.push(databaseName);
    }
  }
  targetDatabaseList.value = databaseNameList;
});

const source = computed((): ChangeHistorySourceSchema | undefined => {
  if (!changeHistory.value) {
    return;
  }
  const { databaseName } = extractDatabaseNameAndChangeHistoryUID(
    changeHistory.value.name
  );
  const database = databaseStore.getDatabaseByName(databaseName);
  return {
    databaseName,
    changeHistory: changeHistory.value,
    environmentName: database.effectiveEnvironment,
    projectName: database.project,
  };
});

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
</script>
