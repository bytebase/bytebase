<template>
  <div v-if="database" class="flex items-center flex-wrap gap-1">
    <InstanceV1Name :instance="database.instanceEntity" :link="false">
      <template #prefix>
        <EnvironmentV1Name
          :environment="database.instanceEntity.environmentEntity"
          :link="false"
          :show-icon="false"
          text-class=" text-control-light"
        />
      </template>
    </InstanceV1Name>

    <div class="flex items-center gap-x-1">
      <heroicons-outline:database />

      <EnvironmentV1Name
        v-if="
          database.instanceEntity.environment !== database.effectiveEnvironment
        "
        :environment="database.effectiveEnvironmentEntity"
        :link="false"
        :show-icon="false"
        text-class=" text-control-light"
      />

      <DatabaseV1Name :database="database" :link="false" />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  DatabaseV1Name,
  InstanceV1Name,
  EnvironmentV1Name,
} from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { Sheet } from "@/types/proto/v1/sheet_service";
import { extractDatabaseResourceName } from "@/utils";

const props = defineProps<{
  sheet: Sheet;
}>();
const databaseStore = useDatabaseV1Store();

const database = computed(() => {
  const { sheet } = props;
  if (!props.sheet.database) return undefined;
  const db = databaseStore.getDatabaseByUID(
    extractDatabaseResourceName(sheet.database).database
  );
  if (db.uid === String(UNKNOWN_ID)) return undefined;
  return db;
});
</script>
