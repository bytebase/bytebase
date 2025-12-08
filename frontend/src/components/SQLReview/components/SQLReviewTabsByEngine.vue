<template>
  <NTabs v-model:value="selectedEngine" :type="'line'" :size="'large'">
    <NTabPane
      v-for="[engine, ruleMap] in sortedData"
      :key="engine"
      :name="engine"
    >
      <template #tab>
        <div class="flex items-center gap-x-2">
          <RichEngineName
            :engine="engine"
            tag="p"
            class="text-center text-sm text-main!"
          />
          <span
            class="items-center text-xs px-1 py-0.5 rounded-full bg-gray-200 text-gray-800"
          >
            {{ ruleMap.size }}
          </span>
        </div>
      </template>
      <template #default>
        <slot :rule-list="[...ruleMap.values()]" :engine="selectedEngine" />
      </template>
    </NTabPane>
  </NTabs>
</template>

<script setup lang="ts">
import { NTabPane, NTabs } from "naive-ui";
import { computed, ref, watch } from "vue";
import { RichEngineName } from "@/components/v2";
import type { RuleTemplateV2 } from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import { supportedEngineV1List } from "@/utils";

const selectedEngine = ref<Engine>(0); // UNSPECIFIED

const props = defineProps<{
  ruleMapByEngine: Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>;
}>();

watch(
  () => [...props.ruleMapByEngine.keys()][0] ?? 0, // UNSPECIFIED
  (engine) => (selectedEngine.value = engine),
  { immediate: true }
);

const engineWithOrderRank = computed(() => {
  return supportedEngineV1List().reduce((map, engine, index) => {
    map.set(engine, index);
    return map;
  }, new Map<Engine, number>());
});

const sortedData = computed(
  (): [Engine, Map<SQLReviewRule_Type, RuleTemplateV2>][] => {
    return [...props.ruleMapByEngine.entries()].sort(([e1], [e2]) => {
      return (
        (engineWithOrderRank.value.get(e1) ?? 0) -
        (engineWithOrderRank.value.get(e2) ?? 0)
      );
    });
  }
);
</script>
