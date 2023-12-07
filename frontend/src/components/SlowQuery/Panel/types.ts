import dayjs from "dayjs";
import { useDatabaseV1Store } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { ListSlowQueriesRequest } from "@/types/proto/v1/database_service";
import { SearchScope } from "@/utils";

export const buildListSlowQueriesRequest = (
  scopes: SearchScope[],
  timeRange: { fromTime: number | undefined; toTime: number | undefined }
) => {
  const request = {} as Partial<ListSlowQueriesRequest>;

  const project = scopes.find((s) => s.id === "project")?.value;
  const environment = scopes.find((s) => s.id === "environment")?.value;
  const instance = scopes.find((s) => s.id === "instance")?.value;
  const database = scopes.find((s) => s.id === "database")?.value;
  const { fromTime, toTime } = timeRange;

  const query: string[] = [];
  request.parent = "instances/-/databases/-";
  if (database) {
    const uid = database.split("-").slice(-1)[0];
    const db = useDatabaseV1Store().getDatabaseByUID(uid);
    if (db.uid !== `${UNKNOWN_ID}`) {
      request.parent = db.name;
    }
  } else if (instance) {
    request.parent = `instances/${instance}/databases/-`;
  } else if (environment) {
    request.parent = `instances/-/databases/-`;
    query.push(`environment = "environments/${environment}"`);
  }

  if (project) {
    query.push(`project = "projects/${project}"`);
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
