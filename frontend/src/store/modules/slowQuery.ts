import { defineStore } from "pinia";
import { uniq, uniqBy } from "lodash-es";

import { databaseServiceClient, instanceServiceClient } from "@/grpcweb";
import {
  ListSlowQueriesRequest,
  SlowQueryLog,
} from "@/types/proto/v1/database_service";
import { ComposedSlowQueryLog, Instance, unknown } from "@/types";
import { useDatabaseStore } from "./database";
import { extractDatabaseResourceName } from "@/utils";

export const useSlowQueryStore = defineStore("slow-query", () => {
  const fetchSlowQueryLogList = async (
    request: Partial<ListSlowQueriesRequest> = {}
  ) => {
    try {
      const response = await databaseServiceClient.listSlowQueries(request);
      const composedLogList = await composeSlowQueryLogDatabase(
        response.slowQueryLogs
      );
      return composedLogList;
    } catch (ex) {
      return [];
    }
  };

  const syncSlowQueriesByInstance = async (instance: Instance) => {
    const name = `environments/${instance.environment.resourceId}/instances/${instance.resourceId}`;
    await instanceServiceClient.syncSlowQueries({
      instance: name,
    });
  };

  return { fetchSlowQueryLogList, syncSlowQueriesByInstance };
});

const composeSlowQueryLogDatabase = async (
  slowQueryLogList: SlowQueryLog[]
) => {
  type ResourceName = {
    environment: string;
    instance: string;
    database: string;
  };

  const resourceNameList = uniq(slowQueryLogList.map((log) => log.resource))
    .map(extractDatabaseResourceName)
    .filter(
      (item) => item.database && item.instance && item.environment
    ) as ResourceName[];
  const instanceNameList = uniqBy(resourceNameList, (item) => item.instance);
  const instanceList = await Promise.all(
    instanceNameList.map((item) =>
      instanceServiceClient.getInstance({
        name: `environments/${item.environment}/instances/${item.instance}`,
      })
    )
  );
  const instanceMap = new Map(
    instanceList.map((instance) => [instance.name, instance])
  );
  const databaseList = await Promise.all(
    resourceNameList.map((resource) => {
      const instanceName = `environments/${resource.environment}/instances/${resource.instance}`;
      const instance = instanceMap.get(instanceName);
      if (!instance) return Promise.resolve(unknown("DATABASE"));
      const request = useDatabaseStore().fetchDatabaseByInstanceIdAndName({
        instanceId: instance.uid,
        name: resource.database,
      });
      return request;
    })
  );
  const databaseMap = new Map(
    databaseList.map((db) => [
      `environments/${db.instance.environment.resourceId}/instances/${db.instance.resourceId}/databases/${db.name}`,
      db,
    ])
  );
  return slowQueryLogList.map<ComposedSlowQueryLog>((log) => ({
    log,
    database: databaseMap.get(log.resource) ?? unknown("DATABASE"), // databaseMap.get(log.resource)!,
  }));
};
