import type { MaybeRef } from "vue";
import { computed, ref, unref, watch } from "vue";
import type { LocationQuery } from "vue-router";
import { useRoute, useRouter } from "vue-router";
import { usePlanStore } from "@/store/modules/v1/plan";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import { emptyPlan, type ComposedPlan } from "@/types/v1/issue/plan";
import { uidFromSlug } from "@/utils";
import { createPlanSkeleton } from "./create";

export * from "./create";

export * from "./util";

export function useInitializePlan(
  planSlug: MaybeRef<string>,
  project: MaybeRef<string> = "-",
  redirectNotFound: boolean = true
) {
  const isCreating = computed(() => {
    return unref(planSlug).toLowerCase() === "create";
  });
  const uid = computed(() => {
    const slug = unref(planSlug);
    if (slug.toLowerCase() === "create") return String(EMPTY_ID);
    const uid = Number(uidFromSlug(slug));
    if (uid > 0) return String(uid);
    return String(UNKNOWN_ID);
  });
  const route = useRoute();
  const router = useRouter();
  const planStore = usePlanStore();
  const isInitializing = ref(false);

  const plan = ref<ComposedPlan>(emptyPlan());

  const runner = async (uid: string, project: string, url: string) => {
    const plan =
      uid === String(EMPTY_ID)
        ? await createPlanSkeleton(
            route,
            convertRouterQuery(router.resolve(url).query)
          )
        : await planStore.fetchPlanByUID(uid, project);
    return {
      plan,
      url,
    };
  };

  watch(
    [uid, () => unref(project)],
    ([uid, project]) => {
      if (uid === String(UNKNOWN_ID) && redirectNotFound) {
        router.push({ name: "error.404" });
        return;
      }
      const url = route.fullPath;
      isInitializing.value = true;
      runner(uid, project, url).then((result) => {
        if (result.url !== route.fullPath) {
          // the url changed, drop the outdated result
          return;
        }
        plan.value = result.plan;
        isInitializing.value = false;
      });
    },
    { immediate: true }
  );

  return { isCreating, plan, isInitializing };
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
