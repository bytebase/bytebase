import dayjs from "dayjs";
import { UNKNOWN_ID } from "@/types";
import {
  Database,
  ListSlowQueriesRequest,
} from "@/types/proto/v1/database_service";
import { Environment } from "@/types/proto/v1/environment_service";
import { Instance } from "@/types/proto/v1/instance_service";
import { Project } from "@/types/proto/v1/project_service";

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
  const { project, environment, instance, database, fromTime, toTime } = filter;

  const query: string[] = [];
  request.parent = "instances/-/databases/-";
  if (database && database.uid !== String(UNKNOWN_ID)) {
    request.parent = database.name;
  } else if (instance && instance.uid !== String(UNKNOWN_ID)) {
    request.parent = `${instance.name}/databases/-`;
  } else if (environment && environment.uid !== String(UNKNOWN_ID)) {
    request.parent = `instances/-/databases/-`;
    query.push(`environment = "${environment.name}"`);
  }

  if (project) {
    query.push(`project = "${project.name}"`);
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
