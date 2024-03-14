<template>
  <label class="flex items-center text-sm h-5 ml-0.5 whitespace-nowrap">
    <EnvironmentV1Name
      v-if="instance.uid !== String(UNKNOWN_ID)"
      :environment="instance.environmentEntity"
      :link="false"
    />
    <template v-if="instance.uid !== String(UNKNOWN_ID)">
      <heroicons-solid:chevron-right class="flex-shrink-0 h-4 w-4 opacity-70" />
      <span>{{ instance.title }}</span>
    </template>
    <template v-if="database.uid !== String(UNKNOWN_ID)">
      <heroicons-solid:chevron-right class="flex-shrink-0 h-4 w-4 opacity-70" />
      <span>{{ database.databaseName }}</span>
    </template>
  </label>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { EnvironmentV1Name } from "@/components/v2";
import { useDatabaseV1Store, useInstanceV1Store } from "@/store";
import type { SQLEditorTab } from "@/types";
import { UNKNOWN_ID } from "@/types";

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

const instance = computed(() => {
  return useInstanceV1Store().getInstanceByName(connection.value.instance);
});
const database = computed(() => {
  return useDatabaseV1Store().getDatabaseByName(connection.value.database);
});
</script>
