<template>
  <label class="flex items-center text-sm gap-x-0.5">
    <template v-if="instance.id !== UNKNOWN_ID">
      <span>{{ instance.environment.name }}</span>
      <ProtectedEnvironmentIcon
        :environment="instance.environment"
        :class="isCurrentTab && '!text-accent'"
      />
    </template>
    <template v-if="instance.id !== UNKNOWN_ID">
      <heroicons-solid:chevron-right
        class="flex-shrink-0 h-4 w-4 text-control-light"
      />
      <span>{{ instance.name }}</span>
    </template>
    <template v-if="database.id !== UNKNOWN_ID">
      <heroicons-solid:chevron-right
        class="flex-shrink-0 h-4 w-4 text-control-light"
      />
      <span>{{ database.name }}</span>
    </template>
  </label>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";

import type { TabInfo } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { useDatabaseById, useInstanceById, useTabStore } from "@/store";

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

const tabStore = useTabStore();
const connection = computed(() => props.tab.connection);

const instance = useInstanceById(computed(() => connection.value.instanceId));
const database = useDatabaseById(computed(() => connection.value.databaseId));

const isCurrentTab = computed(() => props.tab.id === tabStore.currentTabId);
</script>
