import type { Environment, Instance, Database, Project } from "@/types";

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
