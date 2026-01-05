import { create } from "@bufbuild/protobuf";
import { ConnectError, Code } from "@connectrpc/connect";
import { computed, ref, unref, watch, type MaybeRef } from "vue";
import { useRoute, useRouter, type LocationQuery } from "vue-router";
import {
  issueServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { projectNamePrefix, usePlanStore } from "@/store";
import { EMPTY_ID, UNKNOWN_ID } from "@/types";
import {
  GetIssueRequestSchema,
  Issue_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, PlanCheckRun } from "@/types/proto-es/v1/plan_service_pb";
import { GetRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { Rollout, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { getRolloutFromPlan, extractPlanNameFromRolloutName } from "@/utils";
import { emptyPlan } from "@/types/v1/issue/plan";
import { createPlanSkeleton } from "./create";
import {
  WORKSPACE_ROUTE_403,
  WORKSPACE_ROUTE_404,
} from "@/router/dashboard/workspaceRoutes";

export * from "./create";

export * from "./util";

export function useInitializePlan(
  projectId: MaybeRef<string>,
  planId: MaybeRef<string | undefined>,
  issueId?: MaybeRef<string | undefined>,
  legacyRolloutId?: MaybeRef<string | undefined>
) {
  const isCreating = computed(() => {
    const id = unref(planId) || unref(issueId) || unref(legacyRolloutId);
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
    // Otherwise, if legacyRolloutId is provided, we'll fetch the plan from the rollout
    if (unref(legacyRolloutId)) {
      const id = unref(legacyRolloutId)!;
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
      const planUid = uid.substring(8);
      const rolloutRequest = create(GetRolloutRequestSchema, {
        name: `${projectNamePrefix}${projectId}/plans/${planUid}/rollout`,
      });

      const newRollout =
        await rolloutServiceClientConnect.getRollout(rolloutRequest);
      rolloutResult = newRollout;

      // Extract plan name from rollout name
      const planName = extractPlanNameFromRolloutName(rolloutResult.name);
      if (!planName) {
        throw new Error(
          `Rollout ${planUid} does not have a valid plan reference in its name`
        );
      }

      // Fetch the plan using the extracted plan name
      planResult = await planStore.fetchPlanByName(planName);

      // Fetch the associated issue if it exists via the plan
      if (planResult.issue) {
        try {
          const issueRequest = create(GetIssueRequestSchema, {
            name: planResult.issue,
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
        // Issue without plan - allow it to stay in CICD layout for issue-only view
        // This is expected for grant requests, but may indicate a problem for other issue types
        if (issueResult.type !== Issue_Type.GRANT_REQUEST) {
          console.warn(
            `Issue ${issueUid} of type ${issueResult.type} has no associated plan`,
            issueResult
          );
        }
        return {
          plan: emptyPlan(),
          issue: issueResult,
          rollout: undefined,
          url,
        };
      }

      // Fetch the plan using the issue's plan reference
      planResult = await planStore.fetchPlanByName(issueResult.plan);

      if (planResult.hasRollout) {
        try {
          const rolloutName = getRolloutFromPlan(planResult.name);
          const rolloutRequest = create(GetRolloutRequestSchema, {
            name: rolloutName,
          });
          const newRollout =
            await rolloutServiceClientConnect.getRollout(rolloutRequest);
          rolloutResult = newRollout;
        } catch (error) {
          console.error("Failed to fetch rollout:", error);
          // Rollout might not exist yet, that's ok
          rolloutResult = undefined;
        }
      }
    } else {
      // Direct plan ID
      planResult = await planStore.fetchPlanByName(
        `${projectNamePrefix}${projectId}/plans/${uid}`
      );

      // If we have a plan, try to fetch the associated issue if it exists
      if (planResult.issue) {
        const request = create(GetIssueRequestSchema, {
          name: planResult.issue,
        });
        const newIssue = await issueServiceClientConnect.getIssue(request);
        issueResult = newIssue;
      }

      if (planResult.hasRollout) {
        try {
          const rolloutName = getRolloutFromPlan(planResult.name);
          const rolloutRequest = create(GetRolloutRequestSchema, {
            name: rolloutName,
          });
          const newRollout =
            await rolloutServiceClientConnect.getRollout(rolloutRequest);
          rolloutResult = newRollout;
        } catch (error) {
          console.error("Failed to fetch rollout:", error);
          // Rollout might not exist yet, that's ok
          rolloutResult = undefined;
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
    async ([uid, projectId]) => {
      if (uid === String(UNKNOWN_ID)) {
        router.push({ name: WORKSPACE_ROUTE_404 });
        return;
      }
      const url = route.fullPath;
      isInitializing.value = true;

      try {
        const result = await runner(uid, projectId, url);
        if (result.url !== route.fullPath) {
          // the url changed, drop the outdated result
          return;
        }
        plan.value = result.plan;
        issue.value = result.issue;
        rollout.value = result.rollout;
        isInitializing.value = false;
      } catch (error: unknown) {
        // Check Connect error type and handle accordingly
        if (error instanceof ConnectError) {
          if (error.code === Code.NotFound) {
            router.push({ name: WORKSPACE_ROUTE_404 });
          } else if (error.code === Code.PermissionDenied) {
            router.push({ name: WORKSPACE_ROUTE_403 });
          }
          isInitializing.value = false;
          return;
        }

        isInitializing.value = false;
        console.error("Error initializing plan:", error);
        throw error;
      }
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
  for (const key of Object.keys(query)) {
    const value = query[key];
    if (typeof value === "string") {
      kv[key] = value;
    }
  }
  return kv;
};
