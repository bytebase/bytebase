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
          ruleList: RuleListWithCategory[];
        }"
      >
        <div
          class="flex flex-col justify-start items-start md:flex-row md:items-center md:justify-between"
        >
          <SQLReviewLevelFilter
            v-if="!hideLevelFilter"
            :rule-list="ruleListFilteredByCategory"
            :is-checked-level="(level) => params.checkedLevel.has(level)"
            @toggle-checked-level="$emit('toggle-checked-level', $event)"
          />
          <SearchBox
            ref="searchField"
            class="ml-auto mt-2 md:mt-0 md:!max-w-72"
            style="max-width: 100%"
            :value="params.searchText"
            :placeholder="$t('common.filter-by-name')"
            @update:value="$emit('change-search-text', $event)"
          />
        </div>
        <slot :rule-list="filterRuleList(ruleListFilteredByCategory)" />
      </template>
    </SQLReviewCategoryTabFilter>
  </div>
</template>

<script lang="ts" setup>
import { SearchBox } from "@/components/v2";
import type { RuleTemplateV2 } from "@/types";
import { getRuleLocalization } from "@/types";
import type { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import SQLReviewCategoryTabFilter from "./SQLReviewCategoryTabFilter.vue";
import type { RuleListWithCategory } from "./SQLReviewCategoryTabFilter.vue";
import SQLReviewLevelFilter from "./SQLReviewLevelFilter.vue";
import type { SQLRuleFilterParams } from "./useSQLRuleFilter";

const props = defineProps<{
  ruleList: RuleTemplateV2[];
  params: SQLRuleFilterParams;
  hideLevelFilter?: boolean;
}>();

defineEmits<{
  (event: "toggle-checked-level", level: SQLReviewRuleLevel): void;
  (event: "change-category", category: string): void;
  (event: "change-search-text", keyword: string): void;
}>();

const filterRuleList = (
  list: RuleListWithCategory[]
): RuleListWithCategory[] => {
  if (props.params.checkedLevel.size === 0 && !props.params.searchText) {
    return list;
  }
  return list
    .map((item) => {
      return {
        ...item,
        ruleList: item.ruleList.filter(filterRule),
      };
    })
    .filter((item) => item.ruleList.length > 0);
};

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
  const localization = getRuleLocalization(rule.type, rule.engine);
  if (localization.title.toLowerCase().includes(keyword)) return true;
  if (localization.description.toLowerCase().includes(keyword)) return true;
  return false;
};
</script>
