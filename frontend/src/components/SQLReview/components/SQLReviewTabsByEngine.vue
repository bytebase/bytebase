<template>
  <NTabs v-model:value="selectedEngine" :type="'line'">
    <NTabPane v-for="data in engineList" :key="data.engine" :name="data.engine">
      <template #tab>
        <div class="flex items-center space-x-2">
          <EngineIcon
            :engine="engineFromJSON(data.engine)"
            custom-class="ml-1"
          />
          <RichEngineName
            :engine="data.engine"
            tag="p"
            class="text-center text-sm !text-main"
          />
          <span
            class="items-center text-xs px-1 py-0.5 rounded-full bg-gray-200 text-gray-800"
          >
            {{ data.ruleList.length }}
          </span>
        </div>
      </template>
      <template #default>
        <slot :rule-list="data.ruleList" :engine="selectedEngine" />
      </template>
    </NTabPane>
  </NTabs>
</template>

<script setup lang="ts">
import { NTabs, NTabPane } from "naive-ui";
import { ref, watchEffect } from "vue";
import type { RuleTemplateV2 } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { engineFromJSON } from "@/types/proto/v1/common";

type EngineTypeStats = {
  engine: Engine;
  ruleList: RuleTemplateV2[];
};

const selectedEngine = ref<Engine>(Engine.UNRECOGNIZED);
const engineList = ref<EngineTypeStats[]>([]);

const props = defineProps<{
  ruleList: RuleTemplateV2[];
}>();

watchEffect(() => {
  const tmp = props.ruleList.reduce(
    (dict, rule) => {
      if (!dict[rule.engine]) {
        dict[rule.engine] = {
          engine: rule.engine,
          ruleList: [],
        };
      }
      dict[rule.engine].ruleList.push(rule);
      return dict;
    },
    {} as { [id: string]: EngineTypeStats }
  );
  engineList.value = Object.values(tmp);
  if (engineList.value.length > 0) {
    selectedEngine.value = engineList.value[0].engine;
  }
});
</script>
