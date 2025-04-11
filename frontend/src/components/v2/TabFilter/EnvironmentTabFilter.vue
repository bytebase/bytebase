<template>
  <TabFilter
    :value="environment"
    :items="items"
    @update:value="$emit('update:environment', $event as string)"
  >
    <template #label="{ item }">
      <template v-if="item.value === UNKNOWN_ENVIRONMENT_NAME">{{
        item.label
      }}</template>
      <EnvironmentV1Name
        v-else
        :environment="(item as EnvironmentTabFilterItem).environment"
        :link="false"
      />
    </template>
  </TabFilter>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useEnvironmentV1List } from "@/store";
import { formatEnvironmentName, UNKNOWN_ENVIRONMENT_NAME, unknownEnvironment } from "@/types";
import type { Environment } from "@/types/v1/environment";
import { EnvironmentV1Name } from "../Model";
import TabFilter from "./TabFilter.vue";
import type { TabFilterItem } from "./types";

interface EnvironmentTabFilterItem extends TabFilterItem<string> {
  environment: Environment;
}

const props = withDefaults(
  defineProps<{
    environment?: string; // UNKNOWN_ENVIRONMENT_NAME to "ALL"
    includeAll?: boolean;
  }>(),
  {
    environment: UNKNOWN_ENVIRONMENT_NAME,
    includeAll: false,
  }
);

defineEmits<{
  (event: "update:environment", name: string): void;
}>();

const { t } = useI18n();
const environmentList = useEnvironmentV1List();

const items = computed(() => {
  const reversedEnvironmentList = [...environmentList.value].reverse();
  const environmentItems =
    reversedEnvironmentList.map<EnvironmentTabFilterItem>((env) => ({
      value: formatEnvironmentName(env.id),
      label: env.title,
      environment: env,
    }));
  if (props.environment === UNKNOWN_ENVIRONMENT_NAME || props.includeAll) {
    const dummyAll = {
      ...unknownEnvironment(),
      title: t("common.all"),
    };
    environmentItems.unshift({
      value: formatEnvironmentName(dummyAll.id),
      label: dummyAll.title,
      environment: dummyAll,
    });
  }
  return environmentItems;
});
</script>
