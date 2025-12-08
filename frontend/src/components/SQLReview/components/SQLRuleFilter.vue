<template>
  <div class="gap-y-3">
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
          class="mt-2 flex flex-col justify-start items-start md:flex-row md:items-center md:justify-between"
        >
          <SQLReviewLevelFilter
            v-if="!hideLevelFilter"
            :rule-list="ruleListFilteredByCategory"
            :is-checked-level="(level) => params.checkedLevel.has(level)"
            @toggle-checked-level="$emit('toggle-checked-level', $event)"
          />
          <div v-if="supportSelect" class="flex items-center gap-x-2">
            <NCheckbox
              :checked="selectedRuleCount === ruleList.length"
              :indeterminate="
                selectedRuleCount > 0 && selectedRuleCount !== ruleList.length
              "
              @update:checked="$emit('toggle-select-all', $event)"
            />
            <span class="text-xl text-main font-medium">
              {{ $t("sql-review.select-all") }}
            </span>
          </div>
          <SearchBox
            ref="searchField"
            class="ml-auto mt-2 md:mt-0 md:max-w-72!"
            style="max-width: 100%"
            :value="params.searchText"
            :placeholder="$t('common.filter-by-name')"
            @update:value="$emit('change-search-text', $event)"
          />
        </div>
        <NDivider />
        <slot :rule-list="filterRuleList(ruleListFilteredByCategory)" />
      </template>
    </SQLReviewCategoryTabFilter>
  </div>
</template>

<script lang="ts" setup>
import { NCheckbox, NDivider } from "naive-ui";
import { SearchBox } from "@/components/v2";
import type { RuleTemplateV2 } from "@/types";
import { getRuleLocalization, ruleTypeToString } from "@/types";
import { SQLReviewRule_Level } from "@/types/proto-es/v1/review_config_service_pb";
import type { RuleListWithCategory } from "./SQLReviewCategoryTabFilter.vue";
import SQLReviewCategoryTabFilter from "./SQLReviewCategoryTabFilter.vue";
import SQLReviewLevelFilter from "./SQLReviewLevelFilter.vue";
import type { SQLRuleFilterParams } from "./useSQLRuleFilter";

const props = withDefaults(
  defineProps<{
    ruleList: RuleTemplateV2[];
    params: SQLRuleFilterParams;
    hideLevelFilter?: boolean;
    supportSelect?: boolean;
    selectedRuleCount?: number;
  }>(),
  {
    hideLevelFilter: false,
    supportSelect: false,
    selectedRuleCount: 0,
  }
);

defineEmits<{
  (event: "toggle-checked-level", level: SQLReviewRule_Level): void;
  (event: "toggle-select-all", select: boolean): void;
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
  if (ruleTypeToString(rule.type).toLowerCase().includes(keyword)) return true;
  const localization = getRuleLocalization(
    ruleTypeToString(rule.type),
    rule.engine
  );
  if (localization.title.toLowerCase().includes(keyword)) return true;
  if (localization.description.toLowerCase().includes(keyword)) return true;
  return false;
};
</script>
