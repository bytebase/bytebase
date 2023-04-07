<template>
  <TabFilter
    :value="environment"
    :items="items"
    @update:value="$emit('update:environment', $event)"
  >
    <template #label="{ item }">
      <template v-if="item.value === UNKNOWN_ID">{{ item.label }}</template>
      <EnvironmentName v-else :environment="item.environment" :link="false" />
    </template>
  </TabFilter>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { useEnvironmentList } from "@/store";
import { Environment, EnvironmentId, UNKNOWN_ID, unknown } from "@/types";
import { TabFilterItem } from "./types";
import { EnvironmentName } from "../Model";

interface EnvironmentTabFilterItem extends TabFilterItem<EnvironmentId> {
  environment: Environment;
}

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
const environmentList = useEnvironmentList(["NORMAL"]);

const items = computed(() => {
  const environmentItems = environmentList.value.map<EnvironmentTabFilterItem>(
    (env) => ({
      value: env.id,
      label: env.name,
      environment: env,
    })
  );
  if (props.environment === UNKNOWN_ID || props.includeAll) {
    environmentItems.unshift({
      value: UNKNOWN_ID,
      label: t("common.all"),
      environment: unknown("ENVIRONMENT"),
    });
  }
  return environmentItems;
});
</script>
