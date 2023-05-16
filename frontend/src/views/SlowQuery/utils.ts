import type { LocationQuery } from "vue-router";
import { UNKNOWN_ID } from "@/types";
import {
  SlowQueryFilterParams,
  defaultSlowQueryFilterParams,
} from "@/components/SlowQuery";
import {
  useDatabaseStore,
  useEnvironmentV1Store,
  useInstanceStore,
  useProjectStore,
} from "@/store";

export const wrapQueryFromFilterParams = (params: SlowQueryFilterParams) => {
  const query: Record<string, any> = {};
  if (params.project && params.project.id !== UNKNOWN_ID) {
    query.project = params.project.id;
  }
  if (params.environment && params.environment.uid !== String(UNKNOWN_ID)) {
    query.environment = params.environment.uid;
  }
  if (params.instance && params.instance.id !== UNKNOWN_ID) {
    query.instance = params.instance.id;
  }
  if (params.database && params.database.id !== UNKNOWN_ID) {
    query.database = params.database.id;
  }
  if (params.fromTime) {
    if (params.fromTime !== defaultSlowQueryFilterParams().fromTime) {
      query.fromTime = Math.floor(params.fromTime / 1000);
    }
  } else {
    query.fromTime = -1;
  }
  if (params.toTime) {
    if (params.toTime !== defaultSlowQueryFilterParams().toTime) {
      query.toTime = Math.floor(params.toTime / 1000);
    }
  } else {
    query.toTime = -1;
  }
  return query;
};

export const extractSlowQueryLogFilterFromQuery = async (
  query: LocationQuery
) => {
  const params: SlowQueryFilterParams = defaultSlowQueryFilterParams();
  if (query.environment) {
    const id = (query.environment as string) ?? String(UNKNOWN_ID);
    const environment = useEnvironmentV1Store().getEnvironmentByUID(id);
    if (environment && environment.uid !== String(UNKNOWN_ID)) {
      params.environment = environment;
    }
  }
  if (query.project) {
    const id = parseInt(query.project as string, 10) ?? UNKNOWN_ID;
    const project = await useProjectStore().getOrFetchProjectById(id);
    if (project && project.id !== UNKNOWN_ID) {
      params.project = project;
    }
  }
  if (query.instance) {
    const id = parseInt(query.instance as string, 10) ?? UNKNOWN_ID;
    const instance = await useInstanceStore().getOrFetchInstanceById(id);
    if (instance && instance.id !== UNKNOWN_ID) {
      params.instance = instance;
    }
  }
  if (query.database) {
    const id = parseInt(query.database as string, 10) ?? UNKNOWN_ID;
    const database = await useDatabaseStore().getOrFetchDatabaseById(id);
    if (database && database.id !== UNKNOWN_ID) {
      params.database = database;
    }
  }
  if (query.fromTime) {
    const from = parseInt(query.fromTime as string, 10) * 1000 || undefined;
    if (from) {
      params.fromTime = from > 0 ? from : undefined;
    }
  }
  if (query.toTime) {
    const to = parseInt(query.toTime as string, 10) * 1000 || undefined;
    if (to) {
      params.toTime = to > 0 ? to : undefined;
    }
  }
  return params;
};
