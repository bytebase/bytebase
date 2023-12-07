import { computedAsync } from "@vueuse/core";
import { computed, unref, ComputedRef } from "vue";
import { useRoute } from "vue-router";
import {
  useProjectV1Store,
  useDatabaseV1Store,
  experimentalFetchIssueByUID,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { unknownProject, unknownDatabase, UNKNOWN_ID, EMPTY_ID } from "@/types";
import { idFromSlug } from "@/utils";

export const useCurrentProject = (
  params: ComputedRef<{
    projectSlug?: string;
    projectId?: string;
    issueSlug?: string;
    databaseSlug?: string;
    changeHistorySlug?: string;
  }>
) => {
  const route = useRoute();

  const issueUID = computed(() => {
    const slug = unref(params).issueSlug;
    if (!slug) return String(UNKNOWN_ID);
    if (slug.toLowerCase() === "new") return String(EMPTY_ID);
    const uid = Number(idFromSlug(slug));
    if (uid > 0) return String(uid);
    return String(UNKNOWN_ID);
  });

  const database = computed(() => {
    if (unref(params).changeHistorySlug) {
      const parent = `instances/${route.params.instance}/databases/${route.params.database}`;
      return useDatabaseV1Store().getDatabaseByName(parent);
    } else if (unref(params).databaseSlug) {
      return useDatabaseV1Store().getDatabaseByUID(
        String(idFromSlug(unref(params).databaseSlug!))
      );
    }
    return unknownDatabase();
  });

  const project = computedAsync(async () => {
    if (issueUID.value !== String(UNKNOWN_ID)) {
      if (issueUID.value === String(EMPTY_ID)) {
        const projectUID = route.query.project as string;
        if (!projectUID) return unknownProject();
        return await useProjectV1Store().getOrFetchProjectByUID(projectUID);
      }

      const existedIssue = await experimentalFetchIssueByUID(issueUID.value);
      return existedIssue.projectEntity;
    } else if (unref(params).projectSlug) {
      return useProjectV1Store().getProjectByUID(
        String(idFromSlug(unref(params).projectSlug!))
      );
    } else if (unref(params).projectId) {
      return useProjectV1Store().getProjectByName(
        `${projectNamePrefix}${unref(params).projectId}`
      );
    } else if (unref(params).databaseSlug || unref(params).changeHistorySlug) {
      return database.value.projectEntity;
    }
    return unknownProject();
  }, unknownProject());

  return {
    project,
  };
};
