<template>
  <label class="flex items-center text-sm h-5 ml-0.5 whitespace-nowrap">
    <template v-if="!hideEnvironment">
      <EnvironmentV1Name
        :environment="environment"
        :link="false"
      />
      <ChevronRightIcon class="shrink-0 h-4 w-4 opacity-70" />
    </template>
    <template v-if="isValidInstanceName(instance.name)">
      <span>{{ instance.title }}</span>
      <ChevronRightIcon class="shrink-0 h-4 w-4 opacity-70" />
    </template>
    <template v-if="isValidDatabaseName(database.name)">
      <span>{{ databaseName }}</span>
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
import {
  extractDatabaseResourceName,
  getDatabaseEnvironment,
  getInstanceResource,
} from "@/utils";

const props = defineProps<{
  tab: SQLEditorTab;
}>();

const connection = computed(() => props.tab.connection);

const { database } = useDatabaseV1ByName(
  computed(() => connection.value.database)
);

const instance = computed(() => {
  return getInstanceResource(database.value);
});

const environment = computed(() => {
  return getDatabaseEnvironment(database.value);
});

const databaseName = computed(() => {
  return extractDatabaseResourceName(database.value.name).databaseName;
});

const hideEnvironment = computed(() => {
  return environment.value?.id === String(UNKNOWN_ID);
});
</script>
