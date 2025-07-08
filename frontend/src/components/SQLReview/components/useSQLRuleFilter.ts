import { reactive } from "vue";
import { useRoute } from "vue-router";
import { SQLReviewRuleLevel } from "@/types/proto-es/v1/org_policy_service_pb";

export type SQLRuleFilterParams = {
  checkedLevel: Set<SQLReviewRuleLevel>;
  selectedCategory: string;
  searchText: string;
};

export const useSQLRuleFilter = () => {
  const route = useRoute();
  const params = reactive<SQLRuleFilterParams>({
    checkedLevel: new Set([]),
    selectedCategory: route.query.category
      ? (route.query.category as string)
      : "all",
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
    changeCategory(category: string) {
      params.selectedCategory = category;
    },
    changeSearchText(keyword: string) {
      params.searchText = keyword;
    },
    reset() {
      this.changeCategory("all");
      params.checkedLevel = new Set([]);
    },
  };
  return { params, events };
};
