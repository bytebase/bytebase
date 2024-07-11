import { reactive } from "vue";
import { useRoute } from "vue-router";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";

export type SQLRuleFilterParams = {
  checkedLevel: Set<SQLReviewRuleLevel>;
  selectedCategory: string | undefined;
  searchText: string;
};

export const useSQLRuleFilter = () => {
  const route = useRoute();
  const params = reactive<SQLRuleFilterParams>({
    checkedLevel: new Set([]),
    selectedCategory: route.query.category
      ? (route.query.category as string)
      : undefined,
    searchText: "",
  });
  const events = {
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
    reset() {
      this.changeCategory(undefined);
      params.checkedLevel = new Set([]);
    },
  };
  return { params, events };
};
