import { defineStore } from "pinia";

import { databaseServiceClient } from "@/grpcweb";
import { ListSlowQueriesRequest } from "@/types/proto/v1/database_service";
import { Instance } from "@/types";

export const useSlowQueryStore = defineStore("slow-query", () => {
  const fetchSlowQueryLogList = async (
    request: Partial<ListSlowQueriesRequest> = {}
  ) => {
    console.log(
      `fetchSlowQueryLogList(${JSON.stringify(request, null, "  ")})`
    );
    try {
      const response = await databaseServiceClient.listSlowQueries(request);
      return response.slowQueryLogs;
    } catch (ex) {
      return [];
    }
  };

  const syncSlowQueriesByInstance = async (instance: Instance) => {
    const name = `environments/${instance.environment.resourceId}/instances/${instance.resourceId}`;
    console.log("syncSlowQueriesByInstance", name);
  };

  return { fetchSlowQueryLogList, syncSlowQueriesByInstance };
});
