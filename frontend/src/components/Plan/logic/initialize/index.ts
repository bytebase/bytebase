import { create } from "@bufbuild/protobuf";
import { computed, ref, unref, watch, type MaybeRef } from "vue";
import { useRoute, useRouter, type LocationQuery } from "vue-router";
import {
  issueServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import { projectNamePrefix, usePlanStore } from "@/store";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import { GetIssueRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { GetRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { Rollout, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { emptyPlan } from "@/types/v1/issue/plan";
import { issueV1Slug } from "@/utils";
import { createPlanSkeleton } from "./create";

export * from "./create";

export * from "./util";

export function useInitializePlan(
  projectId: MaybeRef<string>,
  planId: MaybeRef<string | undefined>,
  issueId: MaybeRef<string | undefined>,
  rolloutId?: MaybeRef<string | undefined>
) {
  const isCreating = computed(() => {
    const id = unref(planId) || unref(issueId) || unref(rolloutId);
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
    // Otherwise, if rolloutId is provided, we'll fetch the plan from the rollout
    if (unref(rolloutId)) {
      const id = unref(rolloutId)!;
      if (id.toLowerCase() === "create") return String(EMPTY_ID);
      // For rollout-based initialization, return a special marker
      return `rollout:${id}`;
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
  const isInitializing = ref(true);

  const plan = ref<Plan>(emptyPlan());
  const planCheckRuns = ref<PlanCheckRun[]>([]);
  const issue = ref<Issue | undefined>(undefined);
  const rollout = ref<Rollout | undefined>(undefined);
  const taskRuns = ref<TaskRun[]>([]);

  const runner = async (uid: string, projectId: string, url: string) => {
    let planResult: Plan;
    let issueResult: Issue | undefined = undefined;
    let rolloutResult: Rollout | undefined = undefined;

    if (uid === String(EMPTY_ID)) {
      // Creating a new plan
      planResult = await createPlanSkeleton(
        route,
        convertRouterQuery(router.resolve(url).query)
      );
    } else if (uid.startsWith("rollout:")) {
      // Fetch plan from rollout
      const rolloutUid = uid.substring(8);
      const rolloutRequest = create(GetRolloutRequestSchema, {
        name: `${projectNamePrefix}${projectId}/rollouts/${rolloutUid}`,
      });
      const newRollout =
        await rolloutServiceClientConnect.getRollout(rolloutRequest);
      rolloutResult = newRollout;

      if (!rolloutResult.plan) {
        throw new Error(
          `Rollout ${rolloutUid} does not have an associated plan`
        );
      }

      // Fetch the plan using the rollout's plan reference
      planResult = await planStore.fetchPlanByName(rolloutResult.plan);

      // Fetch the associated issue if it exists
      if (rolloutResult.issue) {
        try {
          const issueRequest = create(GetIssueRequestSchema, {
            name: rolloutResult.issue,
          });
          const newIssue =
            await issueServiceClientConnect.getIssue(issueRequest);
          issueResult = newIssue;
        } catch {
          // Issue might not exist or we don't have permission, that's ok
        }
      }
    } else if (uid.startsWith("issue:")) {
      // Fetch plan from issue
      const issueUid = uid.substring(6);
      const request = create(GetIssueRequestSchema, {
        name: `${projectNamePrefix}${projectId}/issues/${issueUid}`,
      });
      const newIssue = await issueServiceClientConnect.getIssue(request);
      issueResult = newIssue;

      if (!issueResult.plan) {
        // Redirect to legacy issue page for issues without plans.
        router.replace({
          name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
          params: {
            projectId,
            issueSlug: issueV1Slug(issueResult.name, issueResult.title),
          },
        });
        return {
          plan: emptyPlan(),
          issue: issueResult,
          rollout: undefined,
          url,
        };
      }

      // Fetch the plan using the issue's plan reference
      planResult = await planStore.fetchPlanByName(issueResult.plan);

      // Fetch the associated rollout if it exists
      if (issueResult.rollout) {
        try {
          const rolloutRequest = create(GetRolloutRequestSchema, {
            name: issueResult.rollout,
          });
          const newRollout =
            await rolloutServiceClientConnect.getRollout(rolloutRequest);
          rolloutResult = newRollout;
        } catch {
          // Rollout might not exist or we don't have permission, that's ok
        }
      }
    } else {
      // Direct plan ID
      planResult = await planStore.fetchPlanByName(
        `${projectNamePrefix}${projectId}/plans/${uid}`
      );

      // If we have a plan, try to fetch the associated issue if it exists
      if (planResult.issue) {
        try {
          const request = create(GetIssueRequestSchema, {
            name: planResult.issue,
          });
          const newIssue = await issueServiceClientConnect.getIssue(request);
          issueResult = newIssue;
        } catch {
          // Issue might not exist or we don't have permission, that's ok
        }
      }

      // If we have a plan, try to fetch the associated rollout if it exists
      if (planResult.rollout) {
        try {
          const rolloutRequest = create(GetRolloutRequestSchema, {
            name: planResult.rollout,
          });
          const newRollout =
            await rolloutServiceClientConnect.getRollout(rolloutRequest);
          rolloutResult = newRollout;
        } catch {
          // Rollout might not exist or we don't have permission, that's ok
        }
      }
    }

    return {
      plan: planResult,
      issue: issueResult,
      rollout: rolloutResult,
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
        rollout.value = result.rollout;
        isInitializing.value = false;
      });
    },
    { immediate: true }
  );

  return {
    isCreating,
    plan,
    planCheckRuns,
    taskRuns,
    issue,
    rollout,
    isInitializing,
  };
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
