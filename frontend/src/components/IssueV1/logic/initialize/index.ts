import { computed, ref, Ref, watch } from "vue";
import {
  useRoute,
  useRouter,
  _RouteLocationBase,
  LocationQuery,
} from "vue-router";
import { experimentalFetchIssueByUID } from "@/store";
import { ComposedIssue, emptyIssue, EMPTY_ID, UNKNOWN_ID } from "@/types";
import { idFromSlug } from "@/utils";
import { createIssueSkeleton } from "./create";

export * from "./create";

export function useInitializeIssue(issueSlug: Ref<string>) {
  const isCreating = computed(() => issueSlug.value.toLowerCase() == "new");
  const uid = computed(() => {
    const slug = issueSlug.value;
    if (slug.toLowerCase() === "new") return String(EMPTY_ID);
    const uid = Number(idFromSlug(slug));
    if (uid > 0) return String(uid);
    return String(UNKNOWN_ID);
  });
  const route = useRoute();
  const router = useRouter();
  const isInitializing = ref(false);

  const issue = ref<ComposedIssue>(emptyIssue());

  const runner = async (uid: string, url: string) => {
    const issue =
      uid === String(EMPTY_ID)
        ? await createIssueSkeleton(
            convertRouterQuery(router.resolve(url).query)
          )
        : await experimentalFetchIssueByUID(uid);
    return {
      issue,
      url,
    };
  };

  watch(
    uid,
    (uid) => {
      if (uid === String(UNKNOWN_ID)) {
        router.push({ name: "error.404" });
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
      const updated = await createIssueSkeleton({ ...query, ...overrides });
      issue.value = updated;
    } catch {
      // Nothing
    }
  };

  return { isCreating, issue, isInitializing, reInitialize };
}

export const convertRouterQuery = (query: LocationQuery) => {
  const kv: Record<string, string> = {};
  for (const key in query) {
    const value = query[key];
    if (typeof value === "string") {
      kv[key] = value;
    }
  }
  return kv;
};
