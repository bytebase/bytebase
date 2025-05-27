import { computedAsync } from "@vueuse/core";
import type { ComputedRef } from "vue";
import { unref } from "vue";
import { useRoute } from "vue-router";
import { useProjectV1Store, useDatabaseV1Store } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { unknownProject, unknownDatabase } from "@/types";

export const useCurrentProject = (
  params: ComputedRef<{
    projectId?: string;
    issueSlug?: string;
    instanceId?: string;
    databaseName?: string;
    changelogId?: string;
  }>
) => {
  const route = useRoute();

  const database = computedAsync(async () => {
    if (unref(params).changelogId) {
      const parent = `${instanceNamePrefix}${route.params.instanceId}/${databaseNamePrefix}${route.params.databaseName}`;
      return await useDatabaseV1Store().getOrFetchDatabaseByName(parent);
    } else if (unref(params).databaseName) {
      return await useDatabaseV1Store().getOrFetchDatabaseByName(
        `${instanceNamePrefix}${
          unref(params).instanceId
        }/${databaseNamePrefix}${unref(params).databaseName}`
      );
    }
    return unknownDatabase();
  }, unknownDatabase());

  const project = computedAsync(async () => {
    if (unref(params).projectId) {
      return await useProjectV1Store().getOrFetchProjectByName(
        `${projectNamePrefix}${unref(params).projectId}`
      );
    } else if (unref(params).databaseName || unref(params).changelogId) {
      return database.value.projectEntity;
    }
    return unknownProject();
  }, unknownProject());

  return {
    project,
    database,
  };
};
