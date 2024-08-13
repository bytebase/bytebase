<template>
  <label class="flex items-center text-sm h-5 ml-0.5 whitespace-nowrap">
    <EnvironmentV1Name
      :environment="database.effectiveEnvironmentEntity"
      :link="false"
    />
    <template v-if="isValidInstanceName(instance.name)">
      <heroicons-solid:chevron-right class="flex-shrink-0 h-4 w-4 opacity-70" />
      <span>{{ instance.title }}</span>
    </template>
    <template v-if="isValidDatabaseName(database.name)">
      <heroicons-solid:chevron-right class="flex-shrink-0 h-4 w-4 opacity-70" />
      <span>{{ database.databaseName }}</span>
    </template>
  </label>
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { computed } from "vue";
import { EnvironmentV1Name } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import type { SQLEditorTab } from "@/types";
import { isValidInstanceName, isValidDatabaseName } from "@/types";

const props = defineProps({
  tab: {
    type: Object as PropType<SQLEditorTab>,
    required: true,
  },
  index: {
    type: Number,
    required: true,
  },
});

const connection = computed(() => props.tab.connection);

const database = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(connection.value.database);
});
const instance = computed(() => {
  return database.value.instanceResource;
});
</script>
