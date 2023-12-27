import { useLocalStorage } from "@vueuse/core";
import { uniq } from "lodash-es";

const useSearchHistory = () => {
  const searchHistory = useLocalStorage<string[]>(
    "sql-editor-search-result-history",
    []
  );

  const appendSearchResult = (databaseName: string) => {
    if (!databaseName) {
      return;
    }
    searchHistory.value = uniq([databaseName, ...searchHistory.value]);
  };

  return {
    searchHistory,
    appendSearchResult,
  };
};

export default useSearchHistory;
