<template>
  <label class="flex items-center text-sm h-5 ml-0.5 whitespace-nowrap">
    <template v-if="!hideEnvironment">
      <EnvironmentV1Name
        :environment="database.effectiveEnvironmentEntity"
        :link="false"
      />
      <ChevronRightIcon class="flex-shrink-0 h-4 w-4 opacity-70" />
    </template>
    <template v-if="isValidInstanceName(instance.name)">
      <span>{{ instance.title }}</span>
      <ChevronRightIcon class="flex-shrink-0 h-4 w-4 opacity-70" />
    </template>
    <template v-if="isValidDatabaseName(database.name)">
      <span>{{ database.databaseName }}</span>
    </template>
  </label>
</template>

<script lang="ts" setup>
import { ChevronRightIcon } from "lucide-vue-next";
import type { PropType } from "vue";
import { computed } from "vue";
import { EnvironmentV1Name } from "@/components/v2";
import { useAppFeature, useDatabaseV1Store } from "@/store";
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

const hideEnvironment = useAppFeature(
  "bb.feature.sql-editor.hide-environments"
);
const connection = computed(() => props.tab.connection);

const database = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(connection.value.database);
});
const instance = computed(() => {
  return database.value.instanceResource;
});
</script>
