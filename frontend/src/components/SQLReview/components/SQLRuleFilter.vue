<template>
  <div class="space-y-5">
    <SQLReviewCategorySummaryFilter
      class="mt-6 mb-4"
      :rule-list="ruleList"
      :is-checked-engine="(engine) => params.checkedEngine.has(engine)"
      :is-checked-level="(level) => params.checkedLevel.has(level)"
      @toggle-checked-engine="$emit('toggle-checked-engine', $event)"
      @toggle-checked-level="$emit('toggle-checked-level', $event)"
    />
    <div
      class="flex justify-between items-center border-t border-control-border pt-4"
    >
      <SQLReviewCategoryTabFilter
        :selected="params.selectedCategory"
        :category-list="categoryFilterList"
        @select="$emit('change-category', $event)"
      />
      <BBTableSearch
        ref="searchField"
        :placeholder="$t('sql-review.search-rule-name')"
        @change-text="$emit('change-search-text', $event)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { convertToCategoryList, RuleTemplate } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import SQLReviewCategorySummaryFilter from "./SQLReviewCategorySummaryFilter.vue";
import type { CategoryFilterItem } from "./SQLReviewCategoryTabFilter.vue";
import SQLReviewCategoryTabFilter from "./SQLReviewCategoryTabFilter.vue";
import { SQLRuleFilterParams } from "./useSQLRuleFilter";

const props = defineProps<{
  ruleList: RuleTemplate[];
  params: SQLRuleFilterParams;
}>();

defineEmits<{
  (event: "toggle-checked-engine", engine: Engine): void;
  (event: "toggle-checked-level", level: SQLReviewRuleLevel): void;
  (event: "change-category", category: string | undefined): void;
  (event: "change-search-text", keyword: string): void;
}>();

const { t } = useI18n();

const categoryFilterList = computed((): CategoryFilterItem[] => {
  return convertToCategoryList(props.ruleList).map((c) => ({
    id: c.id,
    name: t(`sql-review.category.${c.id.toLowerCase()}`),
  }));
});
</script>
