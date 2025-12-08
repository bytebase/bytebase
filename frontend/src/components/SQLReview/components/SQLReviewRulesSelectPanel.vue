<template>
  <Drawer :show="show" @close="$emit('close')">
    <DrawerContent
      :title="$t('sql-review.select-review-rules')"
      class="w-280 max-w-[100vw] relative"
    >
      <template #default>
        <SQLReviewTabsByEngine :rule-map-by-engine="ruleTemplateMapV2">
          <template
            #default="{
              ruleList: ruleListFilteredByEngine,
              engine,
            }: {
              ruleList: RuleTemplateV2[];
              engine: Engine;
            }"
          >
            <SQLRuleTableWithFilter
              :engine="engine"
              :rule-list="ruleListFilteredByEngine"
              :editable="false"
              :hide-level="true"
              :support-select="true"
              :size="'small'"
              :selected-rule-keys="getSelectedRuleKeys(engine)"
              @update:selected-rule-keys="
                (keys: string[]) =>
                  onSelectedRuleKeysUpdateForEngine(engine, keys)
              "
            />
          </template>
        </SQLReviewTabsByEngine>
      </template>
      <template #footer>
        <div class="flex items-center justify-end gap-x-3">
          <NButton @click="$emit('close')">
            {{ $t("common.close") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { NButton } from "naive-ui";
import { Drawer, DrawerContent } from "@/components/v2";
import { type Engine } from "@/types/proto-es/v1/common_pb";
import type { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";
import { ruleTemplateMapV2 } from "@/types/sqlReview";
import SQLReviewTabsByEngine from "./SQLReviewTabsByEngine.vue";
import SQLRuleTableWithFilter from "./SQLRuleTableWithFilter.vue";
import { getRuleKey } from "./utils";

const props = defineProps<{
  show: boolean;
  selectedRuleMap: Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "rule-select", rule: RuleTemplateV2): void;
  (event: "rule-remove", rule: RuleTemplateV2): void;
}>();

const getSelectedRuleKeys = (engine: Engine) => {
  const keys: string[] = [];
  const map = props.selectedRuleMap.get(engine);
  if (!map) {
    return keys;
  }
  for (const rule of map.values()) {
    keys.push(getRuleKey(rule));
  }
  return keys;
};

const onSelectedRuleKeysUpdateForEngine = (engine: Engine, keys: string[]) => {
  const oldSelectedKeys = new Set(getSelectedRuleKeys(engine));
  const newSelectedKeys = new Set(keys);

  for (const key of newSelectedKeys) {
    if (oldSelectedKeys.has(key)) {
      oldSelectedKeys.delete(key);
      continue;
    }
    const [engineStr, ruleKey] = key.split(":");
    const engineNum = parseInt(engineStr) as Engine;
    const ruleType = parseInt(ruleKey) as SQLReviewRule_Type;
    const rule = ruleTemplateMapV2.get(engineNum)?.get(ruleType);
    if (rule) {
      emit("rule-select", rule);
    }
  }

  // keys remained in the oldSelectedKeys is not selected.
  for (const oldKey of oldSelectedKeys) {
    const [engineStr, ruleKey] = oldKey.split(":");
    const engineNum = parseInt(engineStr) as Engine;
    const ruleType = parseInt(ruleKey) as SQLReviewRule_Type;
    const rule = props.selectedRuleMap.get(engineNum)?.get(ruleType);
    if (rule) {
      emit("rule-remove", rule);
    }
  }
};
</script>
