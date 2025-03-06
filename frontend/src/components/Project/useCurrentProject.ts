import { computedAsync } from "@vueuse/core";
import type { ComputedRef } from "vue";
import { computed, unref } from "vue";
import { useRoute } from "vue-router";
import {
  useProjectV1Store,
  useDatabaseV1Store,
  experimentalFetchIssueByUID,
} from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
  projectNamePrefix,
} from "@/store/modules/v1/common";
import { unknownProject, unknownDatabase, UNKNOWN_ID, EMPTY_ID } from "@/types";
import { uidFromSlug } from "@/utils";

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

  const issueUID = computed(() => {
    const slug = unref(params).issueSlug;
    if (!slug) return String(UNKNOWN_ID);
    if (slug.toLowerCase() === "new") return String(EMPTY_ID);
    if (slug.toLowerCase() === "create") return String(EMPTY_ID);
    const uid = Number(uidFromSlug(slug));
    if (uid > 0) return String(uid);
    return String(UNKNOWN_ID);
  });

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
    } else if (issueUID.value !== String(UNKNOWN_ID)) {
      if (issueUID.value === String(EMPTY_ID)) {
        const projectUID = route.query.project as string;
        if (!projectUID) return unknownProject();
        return await useProjectV1Store().getOrFetchProjectByName(
          `projects/${projectUID}`
        );
      }

      const existedIssue = await experimentalFetchIssueByUID(issueUID.value);
      return existedIssue.projectEntity;
    }
    return unknownProject();
  }, unknownProject());

  return {
    project,
    database,
  };
};
