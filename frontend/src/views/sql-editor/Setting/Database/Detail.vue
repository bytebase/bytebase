<template>
  <DatabaseDetail v-bind="databaseDetailProps" />
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { provideDatabaseDetailContext } from "@/components/Database/context";
import { useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import {
  extractDatabaseResourceName,
  extractProjectResourceName,
} from "@/utils";
import DatabaseDetail from "@/views/DatabaseDetail";

const props = defineProps<{
  database: ComposedDatabase;
}>();

const databaseDetailProps = computed(() => {
  const { database } = props;
  const projectId = extractProjectResourceName(database.project);
  const { instanceName: instanceId, databaseName } =
    extractDatabaseResourceName(database.name);
  return { projectId, instanceId, databaseName };
});

const dbSchemaStore = useDBSchemaV1Store();
provideDatabaseDetailContext(
  computed(() => databaseDetailProps.value.instanceId),
  computed(() => databaseDetailProps.value.databaseName)
);

onMounted(async () => {
  await dbSchemaStore.getOrFetchDatabaseMetadata({
    database: props.database.name,
  });
});
</script>
