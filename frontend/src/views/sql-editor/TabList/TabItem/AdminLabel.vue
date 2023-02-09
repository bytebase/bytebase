<template>
  <label class="flex items-center text-sm h-6 ml-0.5 whitespace-nowrap">
    <template v-if="instance.id !== UNKNOWN_ID">
      <span>{{ instance.environment.name }}</span>
      <ProductionEnvironmentIcon
        :environment="instance.environment"
        class="w-4 h-4 ml-0.5 !text-current"
      />
    </template>
    <template v-if="instance.id !== UNKNOWN_ID">
      <heroicons-solid:chevron-right class="flex-shrink-0 h-4 w-4 opacity-70" />
      <span>{{ instance.name }}</span>
    </template>
    <template v-if="database.id !== UNKNOWN_ID">
      <heroicons-solid:chevron-right class="flex-shrink-0 h-4 w-4 opacity-70" />
      <span>{{ database.name }}</span>
    </template>
  </label>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";

import type { TabInfo } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { useDatabaseById, useInstanceById } from "@/store";
import ProductionEnvironmentIcon from "@/components/Environment/ProductionEnvironmentIcon.vue";

const props = defineProps({
  tab: {
    type: Object as PropType<TabInfo>,
    required: true,
  },
  index: {
    type: Number,
    required: true,
  },
});

const connection = computed(() => props.tab.connection);

const instance = useInstanceById(computed(() => connection.value.instanceId));
const database = useDatabaseById(computed(() => connection.value.databaseId));
</script>
