import { computed, ref, unref, watch, type MaybeRef } from "vue";
import { useRoute, useRouter, type LocationQuery } from "vue-router";
import { issueServiceClient } from "@/grpcweb";
import { projectNamePrefix, usePlanStore } from "@/store";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import type { Issue } from "@/types/proto/v1/issue_service";
import type { Plan, PlanCheckRun } from "@/types/proto/v1/plan_service";
import { emptyPlan } from "@/types/v1/issue/plan";
import { createPlanSkeleton } from "./create";

export * from "./create";

export * from "./util";

export function useInitializePlan(
  projectId: MaybeRef<string>,
  planId: MaybeRef<string | undefined>,
  issueId: MaybeRef<string | undefined>
) {
  const isCreating = computed(() => {
    const id = unref(planId) || unref(issueId);
    return id?.toLowerCase() === "create";
  });

  const uid = computed(() => {
    // If planId is provided, use it directly
    if (unref(planId)) {
      const id = unref(planId)!;
      if (id.toLowerCase() === "create") return String(EMPTY_ID);
      const uid = Number(id);
      if (uid > 0) return String(uid);
      return String(UNKNOWN_ID);
    }
    // Otherwise, if issueId is provided, we'll fetch the plan from the issue
    if (unref(issueId)) {
      const id = unref(issueId)!;
      if (id.toLowerCase() === "create") return String(EMPTY_ID);
      // For issue-based initialization, return a special marker
      return `issue:${id}`;
    }
    return String(UNKNOWN_ID);
  });

  const route = useRoute();
  const router = useRouter();
  const planStore = usePlanStore();
  const isInitializing = ref(false);

  const plan = ref<Plan>(emptyPlan());
  const planCheckRunList = ref<PlanCheckRun[]>([]);
  const issue = ref<Issue | undefined>(undefined);

  const runner = async (uid: string, projectId: string, url: string) => {
    let planResult: Plan;
    let issueResult: Issue | undefined = undefined;

    if (uid === String(EMPTY_ID)) {
      // Creating a new plan
      planResult = await createPlanSkeleton(
        route,
        convertRouterQuery(router.resolve(url).query)
      );
    } else if (uid.startsWith("issue:")) {
      // Fetch plan from issue
      const issueUid = uid.substring(6);
      issueResult = await issueServiceClient.getIssue({
        name: `${projectNamePrefix}${projectId}/issues/${issueUid}`,
      });
      if (!issueResult.plan) {
        // Should not happen, but handle gracefully
        throw new Error(`Issue ${issueUid} does not have an associated plan`);
      }

      // Fetch the plan using the issue's plan reference
      planResult = await planStore.fetchPlanByName(issueResult.plan);
    } else {
      // Direct plan ID
      planResult = await planStore.fetchPlanByName(
        `${projectNamePrefix}${projectId}/plans/${uid}`
      );

      // If we have a plan, try to fetch the associated issue if it exists
      if (planResult.issue) {
        try {
          issueResult = await issueServiceClient.getIssue({
            name: planResult.issue,
          });
        } catch {
          // Issue might not exist or we don't have permission, that's ok
        }
      }
    }

    return {
      plan: planResult,
      issue: issueResult,
      url,
    };
  };

  watch(
    [uid, () => unref(projectId)],
    ([uid, projectId]) => {
      if (uid === String(UNKNOWN_ID)) {
        router.push({ name: "error.404" });
        return;
      }
      const url = route.fullPath;
      isInitializing.value = true;
      runner(uid, projectId, url).then(async (result) => {
        if (result.url !== route.fullPath) {
          // the url changed, drop the outdated result
          return;
        }
        plan.value = result.plan;
        issue.value = result.issue;
        isInitializing.value = false;
      });
    },
    { immediate: true }
  );

  return { isCreating, plan, planCheckRunList, issue, isInitializing };
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
