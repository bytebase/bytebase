import { reactive } from "vue";
import { useRoute } from "vue-router";
import { SQLReviewRule_Level } from "@/types/proto-es/v1/review_config_service_pb";

export type SQLRuleFilterParams = {
  checkedLevel: Set<SQLReviewRule_Level>;
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
    toggleCheckedLevel(level: SQLReviewRule_Level) {
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
