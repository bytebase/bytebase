<template>
  <NTabs v-model:value="selectedEngine" :type="'line'" :size="'large'">
    <NTabPane
      v-for="[engine, ruleMap] in sortedData"
      :key="engine"
      :name="engine"
    >
      <template #tab>
        <div class="flex items-center space-x-2">
          <EngineIcon :engine="engine" custom-class="ml-1" />
          <RichEngineName
            :engine="engine"
            tag="p"
            class="text-center text-sm !text-main"
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
import { NTabs, NTabPane } from "naive-ui";
import { ref, watch, computed } from "vue";
import { EngineIcon } from "@/components/Icon";
import { RichEngineName } from "@/components/v2";
import type { RuleTemplateV2 } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { supportedEngineV1List } from "@/utils";

const selectedEngine = ref<Engine>(Engine.UNRECOGNIZED);

const props = defineProps<{
  ruleMapByEngine: Map<Engine, Map<string, RuleTemplateV2>>;
}>();

watch(
  () => [...props.ruleMapByEngine.keys()][0] ?? Engine.UNRECOGNIZED,
  (engine) => (selectedEngine.value = engine),
  { immediate: true }
);

const engineWithOrderRank = computed(() => {
  return supportedEngineV1List().reduce((map, engine, index) => {
    map.set(engine, index);
    return map;
  }, new Map<Engine, number>());
});

const sortedData = computed((): [Engine, Map<string, RuleTemplateV2>][] => {
  return [...props.ruleMapByEngine.entries()].sort(([e1], [e2]) => {
    return (
      (engineWithOrderRank.value.get(e1) ?? 0) -
      (engineWithOrderRank.value.get(e2) ?? 0)
    );
  });
});
</script>
