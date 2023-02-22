import { computed, reactive, unref } from "vue";
import {
  MaybeRef,
  RuleLevel,
  RuleTemplate,
  SchemaRuleEngineType,
} from "@/types";

export type SQLRuleFilterParams = {
  checkedEngine: Set<SchemaRuleEngineType>;
  checkedLevel: Set<RuleLevel>;
  selectedCategory: string | undefined;
  searchText: string;
};

export const useSQLRuleFilter = (ruleList: MaybeRef<RuleTemplate[]>) => {
  const params = reactive<SQLRuleFilterParams>({
    checkedEngine: new Set(),
    checkedLevel: new Set(),
    selectedCategory: undefined,
    searchText: "",
  });
  const events = {
    toggleCheckedEngine(engine: SchemaRuleEngineType) {
      if (params.checkedEngine.has(engine)) {
        params.checkedEngine.delete(engine);
      } else {
        params.checkedEngine.add(engine);
      }
    },
    toggleCheckedLevel(level: RuleLevel) {
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
    return unref(ruleList).filter((rule) => {
      if (
        !params.selectedCategory &&
        !params.searchText &&
        params.checkedEngine.size === 0 &&
        params.checkedLevel.size === 0
      ) {
        // Select "All"
        return true;
      }

      return (
        (!params.selectedCategory ||
          rule.category === params.selectedCategory) &&
        (!params.searchText ||
          rule.type.toLowerCase().includes(params.searchText.toLowerCase())) &&
        (params.checkedEngine.size === 0 ||
          rule.engineList.some((engine) => params.checkedEngine.has(engine))) &&
        (params.checkedLevel.size === 0 || params.checkedLevel.has(rule.level))
      );
    });
  });
  return { params, events, filteredRuleList };
};
