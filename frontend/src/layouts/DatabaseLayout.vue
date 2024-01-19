<template>
  <router-view
    :project-id="projectId"
    :instance-id="instanceId"
    :database-name="databaseName"
  />
</template>

<script lang="ts" setup>
import { onMounted } from "vue";
import { provideDatabaseDetailContext } from "@/components/Database/context";
import { useDBSchemaV1Store, useDatabaseV1Store } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";

const props = defineProps<{
  projectId: string;
  instanceId: string;
  databaseName: string;
}>();

const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
provideDatabaseDetailContext(
  props.projectId,
  props.instanceId,
  props.databaseName
);

onMounted(async () => {
  const database = await databaseStore.getOrFetchDatabaseByName(
    `${instanceNamePrefix}${props.instanceId}/${databaseNamePrefix}${props.databaseName}`
  );
  await dbSchemaStore.getOrFetchDatabaseMetadata({
    database: database.name,
    view: DatabaseMetadataView.DATABASE_METADATA_VIEW_BASIC,
  });
});
</script>
