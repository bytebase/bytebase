import { computed, reactive, unref } from "vue";
import { useRoute } from "vue-router";
import { getRuleLocalization, MaybeRef, RuleTemplate } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";

export type SQLRuleFilterParams = {
  checkedEngine: Set<Engine>;
  checkedLevel: Set<SQLReviewRuleLevel>;
  selectedCategory: string | undefined;
  searchText: string;
};

export const useSQLRuleFilter = (ruleList: MaybeRef<RuleTemplate[]>) => {
  const route = useRoute();
  const params = reactive<SQLRuleFilterParams>({
    checkedEngine: new Set(),
    checkedLevel: new Set(),
    selectedCategory: route.query.category
      ? (route.query.category as string)
      : undefined,
    searchText: "",
  });
  const events = {
    toggleCheckedEngine(engine: Engine) {
      if (params.checkedEngine.has(engine)) {
        params.checkedEngine.delete(engine);
      } else {
        params.checkedEngine.add(engine);
      }
    },
    toggleCheckedLevel(level: SQLReviewRuleLevel) {
      if (params.checkedLevel.has(level)) {
        params.checkedLevel.delete(level);
      } else {
        params.checkedLevel.add(level);
      }
    },
    changeCategory(category: string | undefined) {
      params.selectedCategory = category;
    },
    changeSearchText(keyword: string) {
      params.searchText = keyword;
    },
  };
  const filteredRuleList = computed(() => {
    return unref(ruleList)
      .filter((rule) => {
        if (
          !params.selectedCategory &&
          params.checkedEngine.size === 0 &&
          params.checkedLevel.size === 0
        ) {
          // Select "All"
          return true;
        }

        return (
          (!params.selectedCategory ||
            rule.category === params.selectedCategory) &&
          (params.checkedEngine.size === 0 ||
            rule.engineList.some((engine) =>
              params.checkedEngine.has(engine)
            )) &&
          (params.checkedLevel.size === 0 ||
            params.checkedLevel.has(rule.level))
        );
      })
      .filter((rule) => filterRuleByKeyword(rule, params.searchText));
  });
  return { params, events, filteredRuleList };
};

const filterRuleByKeyword = (rule: RuleTemplate, keyword: string) => {
  keyword = keyword.trim().toLowerCase();
  if (!keyword) return true;
  if (rule.type.toLowerCase().includes(keyword)) return true;
  const localization = getRuleLocalization(rule.type);
  if (localization.title.toLowerCase().includes(keyword)) return true;
  if (localization.description.toLowerCase().includes(keyword)) return true;
  return false;
};
