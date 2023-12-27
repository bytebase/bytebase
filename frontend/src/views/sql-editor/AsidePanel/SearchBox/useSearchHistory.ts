import { useLocalStorage } from "@vueuse/core";
import { uniq } from "lodash-es";

const useSearchHistory = () => {
  const searchHistoryRef = useLocalStorage<string[]>(
    "sql-editor-search-result-history",
    []
  );

  const appendSearchResult = (databaseName: string) => {
    if (!databaseName) {
      return;
    }
    searchHistoryRef.value = uniq([databaseName, ...searchHistoryRef.value]);
  };

  return {
    searchResults: searchHistoryRef.value,
    appendSearchResult,
  };
};

export default useSearchHistory;
