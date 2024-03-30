import { defineStore } from "pinia";
import { ref } from "vue";
import { sqlServiceClient } from "@/grpcweb";
import type { QueryHistory } from "@/types/proto/v1/sql_service";

export const useSQLEditorQueryHistoryStore = defineStore(
  "sqlEditorQueryHistory",
  () => {
    const isFetching = ref(false);
    const queryHistoryList = ref<QueryHistory[]>([]);

    const fetchQueryHistoryList = async () => {
      isFetching.value = true;
      const resp = await sqlServiceClient.searchQueryHistories({
        pageSize: 20,
        filter: `type = "QUERY"`,
      });
      queryHistoryList.value = resp.queryHistories;
      isFetching.value = false;
    };

    return {
      isFetching,
      queryHistoryList,
      fetchQueryHistoryList,
    };
  }
);
