<template>
  <TabFilter
    :value="environment"
    :items="items"
    @update:value="$emit('update:environment', $event)"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { useEnvironmentList } from "@/store";
import { EnvironmentId, UNKNOWN_ID } from "@/types";
import { TabFilterItem } from "./types";

const props = withDefaults(
  defineProps<{
    environment: EnvironmentId; // UNKNOWN_ID(-1) to "ALL"
    includeAll?: boolean;
  }>(),
  {
    includeAll: false,
  }
);

defineEmits<{
  (event: "update:environment", id: EnvironmentId): void;
}>();

const { t } = useI18n();
const environmentList = useEnvironmentList();

const items = computed(() => {
  const environmentItems = environmentList.value.map<
    TabFilterItem<EnvironmentId>
  >((env) => ({
    value: env.id,
    label: env.name,
  }));
  if (props.environment === UNKNOWN_ID || props.includeAll) {
    environmentItems.unshift({
      value: UNKNOWN_ID,
      label: t("common.all"),
    });
  }
  return environmentItems;
});
</script>
