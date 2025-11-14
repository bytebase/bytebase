<template>
  <label class="flex items-center text-sm h-5 ml-0.5 whitespace-nowrap">
    <template v-if="!hideEnvironment">
      <EnvironmentV1Name
        :environment="database.effectiveEnvironmentEntity"
        :link="false"
      />
      <ChevronRightIcon class="shrink-0 h-4 w-4 opacity-70" />
    </template>
    <template v-if="isValidInstanceName(instance.name)">
      <span>{{ instance.title }}</span>
      <ChevronRightIcon class="shrink-0 h-4 w-4 opacity-70" />
    </template>
    <template v-if="isValidDatabaseName(database.name)">
      <span>{{ database.databaseName }}</span>
    </template>
  </label>
</template>

<script lang="ts" setup>
import { ChevronRightIcon } from "lucide-vue-next";
import { computed } from "vue";
import { EnvironmentV1Name } from "@/components/v2";
import { useDatabaseV1ByName } from "@/store";
import {
  isValidDatabaseName,
  isValidInstanceName,
  type SQLEditorTab,
  UNKNOWN_ID,
} from "@/types";

const props = defineProps<{
  tab: SQLEditorTab;
}>();

const connection = computed(() => props.tab.connection);

const { database } = useDatabaseV1ByName(
  computed(() => connection.value.database)
);

const instance = computed(() => {
  return database.value.instanceResource;
});

const hideEnvironment = computed(() => {
  return database.value.effectiveEnvironmentEntity?.id === String(UNKNOWN_ID);
});
</script>
