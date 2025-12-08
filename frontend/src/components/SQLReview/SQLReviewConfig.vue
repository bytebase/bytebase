<template>
  <div v-if="ruleMapByEngine.size > 0">
    <div class="flex justify-end">
      <NButton type="primary" @click="showRuleSelectPanel = true">
        {{ $t("sql-review.add-or-remove-rules") }}
      </NButton>
    </div>
    <SQLReviewTabsByEngine :rule-map-by-engine="ruleMapByEngine">
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
          :editable="true"
          @rule-upsert="onRuleChange"
          @rule-remove="$emit('rule-remove', $event)"
        />
      </template>
    </SQLReviewTabsByEngine>
  </div>
  <NEmpty v-else class="py-12 border rounded-sm">
    <template #extra>
      <NButton type="primary" @click="showRuleSelectPanel = true">
        {{ $t("sql-review.add-rules") }}
      </NButton>
    </template>
  </NEmpty>
  <SQLReviewRulesSelectPanel
    :show="showRuleSelectPanel"
    :selected-rule-map="ruleMapByEngine"
    @close="showRuleSelectPanel = false"
    @rule-select="$emit('rule-upsert', $event, {})"
    @rule-remove="$emit('rule-remove', $event)"
  />
</template>

<script lang="ts" setup>
import { NButton, NEmpty } from "naive-ui";
import { ref } from "vue";
import type { Engine } from "@/types/proto-es/v1/common_pb";
import { SQLReviewRule_Type } from "@/types/proto-es/v1/review_config_service_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";
import SQLReviewRulesSelectPanel from "./components/SQLReviewRulesSelectPanel.vue";
import SQLReviewTabsByEngine from "./components/SQLReviewTabsByEngine.vue";
import SQLRuleTableWithFilter from "./components/SQLRuleTableWithFilter.vue";

defineProps<{
  ruleMapByEngine: Map<Engine, Map<SQLReviewRule_Type, RuleTemplateV2>>;
}>();

const emit = defineEmits<{
  (
    event: "rule-upsert",
    rule: RuleTemplateV2,
    update: Partial<RuleTemplateV2>
  ): void;
  (event: "rule-remove", rule: RuleTemplateV2): void;
}>();

const onRuleChange = (
  rule: RuleTemplateV2,
  overrides: Partial<RuleTemplateV2>
) => {
  emit("rule-upsert", rule, overrides);
};

const showRuleSelectPanel = ref<boolean>(false);
</script>
