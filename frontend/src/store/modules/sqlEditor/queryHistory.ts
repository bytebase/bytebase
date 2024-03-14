import { defineStore } from "pinia";
import { ref } from "vue";
import {
  ActivitySQLEditorQueryPayload,
  SQLEditorQueryHistory,
  UNKNOWN_ID,
} from "@/types";
import { emptySQLEditorConnection } from "@/utils";
import {
  useActivityV1Store,
  useDatabaseV1Store,
  useInstanceV1Store,
} from "../v1";

export const useSQLEditorQueryHistoryStore = defineStore(
  "sqlEditorQueryHistory",
  () => {
    const isFetching = ref(false);
    const queryHistoryList = ref<SQLEditorQueryHistory[]>([]);

    const fetchQueryHistoryList = async () => {
      isFetching.value = true;
      const activityList =
        await useActivityV1Store().fetchActivityListForQueryHistory({
          limit: 20,
          order: "desc",
        });
      const historyList = activityList.map<SQLEditorQueryHistory>(
        (activity) => {
          const payload = JSON.parse(
            activity.payload
          ) as ActivitySQLEditorQueryPayload;

          const connection = emptySQLEditorConnection();
          const database = useDatabaseV1Store().getDatabaseByUID(
            String(payload.databaseId) || String(UNKNOWN_ID)
          );
          if (database.uid !== String(UNKNOWN_ID)) {
            connection.instance = database.instance;
            connection.database = database.name;
          } else {
            const instance = useInstanceV1Store().getInstanceByUID(
              String(payload.instanceId) || String(UNKNOWN_ID)
            );
            if (instance.uid !== String(UNKNOWN_ID))
              connection.instance = instance.name;
          }
          return {
            name: activity.name,
            creator: activity.creator,
            createTime: activity.createTime ?? new Date(),
            statement: payload.statement,
            durationNs: payload.durationNs,
            instance: connection.instance,
            database: connection.database,
            error: payload.error,
          };
        }
      );
      isFetching.value = false;
      queryHistoryList.value = historyList;
    };

    return {
      isFetching,
      queryHistoryList,
      fetchQueryHistoryList,
    };
  }
);
