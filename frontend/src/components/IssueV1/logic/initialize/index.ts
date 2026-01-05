import type { MaybeRef } from "vue";
import { computed, ref, unref, watch } from "vue";
import type { LocationQuery } from "vue-router";
import { useRoute, useRouter } from "vue-router";
import {
  experimentalFetchIssueByUID,
  useCurrentUserV1,
  extractUserId,
} from "@/store";
import { emptyIssue, EMPTY_ID, UNKNOWN_ID, type ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { uidFromSlug, hasProjectPermissionV2 } from "@/utils";
import { createIssueSkeleton } from "./create";
import { WORKSPACE_ROUTE_404 } from "@/router/dashboard/workspaceRoutes";

export * from "./create";

export function useInitializeIssue(
  issueSlug: MaybeRef<string>,
  project: MaybeRef<Project>,
  redirectNotFound: boolean = true
) {
  const isCreating = computed(() => {
    return (
      unref(issueSlug).toLowerCase() == "new" ||
      unref(issueSlug).toLowerCase() === "create"
    );
  });
  const uid = computed(() => {
    const slug = unref(issueSlug);
    if (slug.toLowerCase() === "new") return String(EMPTY_ID);
    if (slug.toLowerCase() === "create") return String(EMPTY_ID);
    const uid = Number(uidFromSlug(slug));
    if (uid > 0) return String(uid);
    return String(UNKNOWN_ID);
  });
  const route = useRoute();
  const router = useRouter();
  const isInitializing = ref(false);
  const currentUser = useCurrentUserV1();

  const issue = ref<ComposedIssue>(emptyIssue());

  const runner = async (uid: string, url: string) => {
    const issue =
      uid === String(EMPTY_ID)
        ? await createIssueSkeleton(
            route,
            convertRouterQuery(router.resolve(url).query)
          )
        : await experimentalFetchIssueByUID(uid, unref(project).name);
    return {
      issue,
      url,
    };
  };

  watch(
    [uid],
    ([uid]) => {
      if (uid === String(UNKNOWN_ID) && redirectNotFound) {
        router.push({ name: WORKSPACE_ROUTE_404 });
        return;
      }
      const url = route.fullPath;
      isInitializing.value = true;
      runner(uid, url).then((result) => {
        if (result.url !== route.fullPath) {
          // the url changed, drop the outdated result
          return;
        }
        issue.value = result.issue;
        isInitializing.value = false;
      });
    },
    { immediate: true }
  );

  const reInitialize = async (overrides: Record<string, string> = {}) => {
    const url = route.fullPath;
    const query = convertRouterQuery(router.resolve(url).query);
    try {
      const updated = await createIssueSkeleton(route, {
        ...query,
        ...overrides,
      });
      issue.value = updated;
    } catch {
      // Nothing
    }
  };

  const allowChange = computed(() => {
    if (isCreating.value) {
      return hasProjectPermissionV2(unref(project), "bb.issues.create");
    }

    if (issue.value.status !== IssueStatus.OPEN) {
      return false;
    }

    if (extractUserId(issue.value.creator) === currentUser.value.email) {
      // Allowed if current user is the creator.
      return true;
    }

    return hasProjectPermissionV2(unref(project), "bb.issues.update");
  });

  return { isCreating, issue, isInitializing, reInitialize, allowChange };
}

export const convertRouterQuery = (query: LocationQuery) => {
  const kv: Record<string, string> = {};
  for (const key of Object.keys(query)) {
    const value = query[key];
    if (typeof value === "string") {
      kv[key] = value;
    }
  }
  return kv;
};
