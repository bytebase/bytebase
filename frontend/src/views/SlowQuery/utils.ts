import type { LocationQuery } from "vue-router";
import {
  SlowQueryFilterParams,
  defaultSlowQueryFilterParams,
} from "@/components/SlowQuery";
import {
  useDatabaseV1Store,
  useEnvironmentV1Store,
  useInstanceV1Store,
  useProjectV1Store,
} from "@/store";
import { UNKNOWN_ID } from "@/types";

export const wrapQueryFromFilterParams = (params: SlowQueryFilterParams) => {
  const query: Record<string, any> = {};
  if (params.project && params.project.uid !== String(UNKNOWN_ID)) {
    query.project = params.project.uid;
  }
  if (params.environment && params.environment.uid !== String(UNKNOWN_ID)) {
    query.environment = params.environment.uid;
  }
  if (params.instance && params.instance.uid !== String(UNKNOWN_ID)) {
    query.instance = params.instance.uid;
  }
  if (params.database && params.database.uid !== String(UNKNOWN_ID)) {
    query.database = params.database.uid;
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
    const id = (query.project as string) ?? String(UNKNOWN_ID);
    const project = await useProjectV1Store().getOrFetchProjectByUID(id);
    if (project && project.uid !== String(UNKNOWN_ID)) {
      params.project = project;
    }
  }
  if (query.instance) {
    const uid = (query.instance as string) ?? UNKNOWN_ID;
    const instance = await useInstanceV1Store().getOrFetchInstanceByUID(uid);
    if (instance && instance.uid !== String(UNKNOWN_ID)) {
      params.instance = instance;
    }
  }
  if (query.database) {
    const uid = (query.database as string) ?? UNKNOWN_ID;
    const database = await useDatabaseV1Store().getOrFetchDatabaseByUID(uid);
    if (database && database.uid !== String(UNKNOWN_ID)) {
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
