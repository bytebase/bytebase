import dayjs from "dayjs";
import { Environment, Instance, Database, Project, UNKNOWN_ID } from "@/types";
import { ListSlowQueriesRequest } from "@/types/proto/v1/database_service";

export type SlowQueryFilterParams = {
  project: Project | undefined; // undefined to "All"
  environment: Environment | undefined; // undefined to "All"
  instance: Instance | undefined; // undefined to "All"
  database: Database | undefined; // undefined to "All"
  timeRange: [number, number] | undefined;
};

export const FilterTypeList = [
  "mode",
  "project",
  "environment",
  "instance",
  "database",
  "time-range",
] as const;

export type FilterType = typeof FilterTypeList[number];

export const defaultSlowQueryFilterParams = (): SlowQueryFilterParams => {
  const now = dayjs();
  const recentWeek: [number, number] = [
    now.subtract(7, "days").startOf("day").valueOf(),
    now.endOf("day").valueOf(),
  ];
  return {
    project: undefined,
    environment: undefined,
    instance: undefined,
    database: undefined,
    timeRange: recentWeek,
  };
};

export const buildListSlowQueriesRequest = (filter: SlowQueryFilterParams) => {
  const request = {} as Partial<ListSlowQueriesRequest>;
  const { project, environment, instance, database, timeRange } = filter;

  request.parent = "environments/-/instances/-/databases/-";
  if (database && database.id !== UNKNOWN_ID) {
    request.parent = `environments/${database.instance.environment.resourceId}/instances/${database.instance.resourceId}/databases/${database.name}`;
  } else if (instance && instance.id !== UNKNOWN_ID) {
    request.parent = `environments/${instance.environment.resourceId}/instances/${instance.resourceId}/databases/-`;
  } else if (environment && environment.id !== UNKNOWN_ID) {
    request.parent = `environments/${environment.resourceId}/instances/-/databases/-`;
  }

  const query: string[] = [];
  if (project) {
    query.push(`project = "projects/${project.resourceId}"`);
  }
  if (timeRange) {
    const start = dayjs(timeRange[0]).startOf("day").toISOString();
    const end = dayjs(timeRange[1]).endOf("day").toISOString();
    query.push(`start_time >= "${start}"`);
    query.push(`start_time <= "${end}"`);
  }
  if (query.length > 0) {
    request.filter = query.join(" && ");
  }

  request.orderBy = "count desc";
  return ListSlowQueriesRequest.fromJSON(request);
};
