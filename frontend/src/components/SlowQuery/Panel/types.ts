import dayjs from "dayjs";
import { useDatabaseV1Store } from "@/store";
import { isValidDatabaseName } from "@/types";
import { ListSlowQueriesRequest } from "@/types/proto/v1/database_service";
import { getTsRangeFromSearchParams, type SearchParams } from "@/utils";

export const buildListSlowQueriesRequest = (searchParams: SearchParams) => {
  const request = {} as Partial<ListSlowQueriesRequest>;

  const { scopes } = searchParams;
  const project = scopes.find((s) => s.id === "project")?.value;
  const environment = scopes.find((s) => s.id === "environment")?.value;
  const database = scopes.find((s) => s.id === "database")?.value;
  const createdTsRange = getTsRangeFromSearchParams(searchParams, "created");

  const query: string[] = [];
  request.parent = `projects/${project}`;
  if (database) {
    const db = useDatabaseV1Store().getDatabaseByName(database);
    if (isValidDatabaseName(db.name)) {
      query.push(`database = "${db.name}"`);
    }
  } else if (environment) {
    query.push(`environment = "environments/${environment}"`);
  }
  if (createdTsRange) {
    const start = dayjs(createdTsRange[0]).toISOString();
    const end = dayjs(createdTsRange[1]).toISOString();
    query.push(`start_time >= "${start}"`);
    query.push(`start_time <= "${end}"`);
  }
  if (query.length > 0) {
    request.filter = query.join(" && ");
  }

  request.orderBy = "maximum_query_time desc";
  return ListSlowQueriesRequest.fromJSON(request);
};
