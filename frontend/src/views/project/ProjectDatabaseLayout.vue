<template>
  <router-view
    :project-id="projectId"
    :instance-id="instanceId"
    :database-name="databaseName"
  />
</template>

<script lang="ts" setup>
import { computed, onMounted } from "vue";
import { provideDatabaseDetailContext } from "@/components/Database/context";
import { useDatabaseV1Store, useDBSchemaV1Store } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
  instanceId: string;
  databaseName: string;
}>();

const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
provideDatabaseDetailContext(
  computed(() => props.instanceId),
  computed(() => props.databaseName)
);

onMounted(async () => {
  const database = await databaseStore.getOrFetchDatabaseByName(
    `${instanceNamePrefix}${props.instanceId}/${databaseNamePrefix}${props.databaseName}`
  );
  await dbSchemaStore.getOrFetchDatabaseMetadata({
    database: database.name,
  });
});
</script>
