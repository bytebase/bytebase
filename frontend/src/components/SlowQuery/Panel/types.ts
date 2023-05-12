import dayjs from "dayjs";
import { Environment, Instance, Database, Project, UNKNOWN_ID } from "@/types";
import { ListSlowQueriesRequest } from "@/types/proto/v1/database_service";

export type SlowQueryFilterParams = {
  project: Project | undefined; // undefined to "All"
  environment: Environment | undefined; // undefined to "All"
  instance: Instance | undefined; // undefined to "All"
  database: Database | undefined; // undefined to "All"
  fromTime: number | undefined;
  toTime: number | undefined;
};

export const FilterTypeList = [
  "project",
  "environment",
  "instance",
  "database",
  "time-range",
] as const;

export type FilterType = typeof FilterTypeList[number];

export const defaultSlowQueryFilterParams = (): SlowQueryFilterParams => {
  const now = dayjs();
  const aWeekAgo = now.subtract(7, "days").startOf("day").valueOf();
  const tonight = now.endOf("day").valueOf();
  return {
    project: undefined,
    environment: undefined,
    instance: undefined,
    database: undefined,
    fromTime: aWeekAgo,
    toTime: tonight,
  };
};

export const buildListSlowQueriesRequest = (filter: SlowQueryFilterParams) => {
  const request = {} as Partial<ListSlowQueriesRequest>;
  const { project, instance, database, fromTime, toTime } = filter;

  request.parent = "instances/-/databases/-";
  if (database && database.id !== UNKNOWN_ID) {
    request.parent = `instances/${database.instance.resourceId}/databases/${database.name}`;
  } else if (instance && instance.id !== UNKNOWN_ID) {
    request.parent = `instances/${instance.resourceId}/databases/-`;
  }

  // TODO(d): fix environment filter.
  const query: string[] = [];
  if (project) {
    query.push(`project = "projects/${project.resourceId}"`);
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
