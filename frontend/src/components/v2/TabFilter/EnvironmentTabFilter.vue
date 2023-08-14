<template>
  <TabFilter
    :value="environment"
    :items="items"
    @update:value="$emit('update:environment', $event)"
  >
    <template #label="{ item }">
      <template v-if="item.value === UNKNOWN_ID">{{ item.label }}</template>
      <EnvironmentV1Name v-else :environment="item.environment" :link="false" />
    </template>
  </TabFilter>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useEnvironmentV1List } from "@/store";
import { UNKNOWN_ID, unknownEnvironment } from "@/types";
import { Environment } from "@/types/proto/v1/environment_service";
import { EnvironmentV1Name } from "../Model";
import { TabFilterItem } from "./types";

interface EnvironmentTabFilterItem extends TabFilterItem<string> {
  environment: Environment;
}

const props = withDefaults(
  defineProps<{
    environment: string; // UNKNOWN_ID(-1) to "ALL"
    includeAll?: boolean;
  }>(),
  {
    includeAll: false,
  }
);

defineEmits<{
  (event: "update:environment", id: string): void;
}>();

const { t } = useI18n();
const environmentList = useEnvironmentV1List(false /* !showDeleted */);

const items = computed(() => {
  const reversedEnvironmentList = [...environmentList.value].reverse();
  const environmentItems =
    reversedEnvironmentList.map<EnvironmentTabFilterItem>((env) => ({
      value: env.uid,
      label: env.title,
      environment: env,
    }));
  if (props.environment === String(UNKNOWN_ID) || props.includeAll) {
    const dummyAll = {
      ...unknownEnvironment(),
      title: t("common.all"),
    };
    environmentItems.unshift({
      value: dummyAll.uid,
      label: dummyAll.title,
      environment: dummyAll,
    });
  }
  return environmentItems;
});
</script>
