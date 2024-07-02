<template>
  <div class="space-y-3">
    <SQLReviewCategoryTabFilter
      :value="params.selectedCategory"
      :rule-list="ruleList"
      @update:value="$emit('change-category', $event)"
    >
      <template
        #default="{
          ruleList: ruleListFilteredByCategory,
        }: {
          ruleList: RuleTemplateV2[];
        }"
      >
        <div class="flex items-center justify-between">
          <SQLReviewCategorySummaryFilter
            :rule-list="ruleListFilteredByCategory"
            :is-checked-level="(level) => params.checkedLevel.has(level)"
            @toggle-checked-level="$emit('toggle-checked-level', $event)"
          />
          <SearchBox
            ref="searchField"
            :value="params.searchText"
            :placeholder="$t('common.filter-by-name')"
            @update:value="$emit('change-search-text', $event)"
          />
        </div>
        <slot :rule-list="ruleListFilteredByCategory.filter(filterRule)" />
      </template>
    </SQLReviewCategoryTabFilter>
  </div>
</template>

<script lang="ts" setup>
import type { RuleTemplateV2 } from "@/types";
import { getRuleLocalization } from "@/types";
import type { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import SQLReviewCategorySummaryFilter from "./SQLReviewCategorySummaryFilter.vue";
import SQLReviewCategoryTabFilter from "./SQLReviewCategoryTabFilter.vue";
import type { SQLRuleFilterParams } from "./useSQLRuleFilter";

const props = defineProps<{
  ruleList: RuleTemplateV2[];
  params: SQLRuleFilterParams;
}>();

defineEmits<{
  (event: "toggle-checked-level", level: SQLReviewRuleLevel): void;
  (event: "change-category", category: string | undefined): void;
  (event: "change-search-text", keyword: string): void;
}>();

const filterRule = (rule: RuleTemplateV2) => {
  if (props.params.checkedLevel.size > 0) {
    if (!props.params.checkedLevel.has(rule.level)) {
      return false;
    }
  }
  return filterRuleByKeyword(rule, props.params.searchText);
};

const filterRuleByKeyword = (rule: RuleTemplateV2, searchText: string) => {
  const keyword = searchText.trim().toLowerCase();
  if (!keyword) return true;
  if (rule.type.toLowerCase().includes(keyword)) return true;
  const localization = getRuleLocalization(rule.type);
  if (localization.title.toLowerCase().includes(keyword)) return true;
  if (localization.description.toLowerCase().includes(keyword)) return true;
  return false;
};
</script>
