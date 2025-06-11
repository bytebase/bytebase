import type { MaybeRef } from "vue";
import { computed, ref, unref, watch } from "vue";
import type { LocationQuery } from "vue-router";
import { useRoute, useRouter } from "vue-router";
import { usePlanStore } from "@/store/modules/v1/plan";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import type { Plan, PlanCheckRun } from "@/types/proto/v1/plan_service";
import { emptyPlan } from "@/types/v1/issue/plan";
import { createPlanSkeleton } from "./create";

export * from "./create";

export * from "./util";

export function useInitializePlan(
  planId: MaybeRef<string>,
  project: MaybeRef<string> = "-",
  redirectNotFound: boolean = true
) {
  const isCreating = computed(() => {
    return unref(planId).toLowerCase() === "create";
  });
  const uid = computed(() => {
    const id = unref(planId);
    if (id.toLowerCase() === "create") return String(EMPTY_ID);
    const uid = Number(id);
    if (uid > 0) return String(uid);
    return String(UNKNOWN_ID);
  });
  const route = useRoute();
  const router = useRouter();
  const planStore = usePlanStore();
  const isInitializing = ref(false);

  const plan = ref<Plan>(emptyPlan());
  const planCheckRunList = ref<PlanCheckRun[]>([]);

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
      runner(uid, project, url).then(async (result) => {
        if (result.url !== route.fullPath) {
          // the url changed, drop the outdated result
          return;
        }
        plan.value = result.plan;
        // Fetch plan check runs if not creating
        if (uid !== String(EMPTY_ID)) {
          const { fetchPlanCheckRuns } = usePlanStore();
          planCheckRunList.value = await fetchPlanCheckRuns(result.plan);
        }
        isInitializing.value = false;
      });
    },
    { immediate: true }
  );

  return { isCreating, plan, planCheckRunList, isInitializing };
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
