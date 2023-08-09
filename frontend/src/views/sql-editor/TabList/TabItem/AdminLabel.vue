<template>
  <label class="flex items-center text-sm h-6 ml-0.5 whitespace-nowrap">
    <EnvironmentV1Name
      v-if="instance.uid !== String(UNKNOWN_ID)"
      :environment="instance.environmentEntity"
      :link="false"
    />
    <template v-if="instance.uid !== String(UNKNOWN_ID)">
      <heroicons-solid:chevron-right class="flex-shrink-0 h-4 w-4 opacity-70" />
      <span>{{ instance.title }}</span>
    </template>
    <template v-if="databaseV1.uid !== String(UNKNOWN_ID)">
      <heroicons-solid:chevron-right class="flex-shrink-0 h-4 w-4 opacity-70" />
      <span>{{ databaseV1.databaseName }}</span>
    </template>
  </label>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { EnvironmentV1Name } from "@/components/v2";
import { useDatabaseV1ByUID, useInstanceV1ByUID } from "@/store";
import type { TabInfo } from "@/types";
import { UNKNOWN_ID } from "@/types";

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

const { instance } = useInstanceV1ByUID(
  computed(() => connection.value.instanceId)
);

const { database: databaseV1 } = useDatabaseV1ByUID(
  computed(() => String(connection.value.databaseId))
);
</script>
