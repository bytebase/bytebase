import dayjs from "dayjs";
import { useDatabaseV1Store } from "@/store";
import { unknownDatabase } from "@/types";
import { ListSlowQueriesRequest } from "@/types/proto/v1/database_service";
import type { SearchScope } from "@/utils";

export const buildListSlowQueriesRequest = (
  scopes: SearchScope[],
  timeRange: { fromTime: number | undefined; toTime: number | undefined }
) => {
  const request = {} as Partial<ListSlowQueriesRequest>;

  const project = scopes.find((s) => s.id === "project")?.value;
  const environment = scopes.find((s) => s.id === "environment")?.value;
  const database = scopes.find((s) => s.id === "database")?.value;
  const { fromTime, toTime } = timeRange;

  const query: string[] = [];
  request.parent = `projects/${project}`;
  if (database) {
    const db = useDatabaseV1Store().getDatabaseByName(database);
    if (db.name !== unknownDatabase().name) {
      query.push(`database = "${db.name}"`);
    }
  } else if (environment) {
    query.push(`environment = "environments/${environment}"`);
  }

  if (fromTime) {
    const start = dayjs(fromTime).toISOString();
    query.push(`start_time >= "${start}"`);
  }
  if (toTime) {
    const end = dayjs(toTime).toISOString();
    query.push(`start_time <= "${end}"`);
  }
  if (query.length > 0) {
    request.filter = query.join(" && ");
  }

  request.orderBy = "maximum_query_time desc";
  return ListSlowQueriesRequest.fromJSON(request);
};
