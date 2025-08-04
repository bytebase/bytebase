import { create } from "@bufbuild/protobuf";
import { includes, uniq } from "lodash-es";
import { v4 as uuidv4 } from "uuid";
import {
  computed,
  inject,
  provide,
  ref,
  watch,
  watchEffect,
  type InjectionKey,
} from "vue";
import { useRoute } from "vue-router";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
} from "@/router/dashboard/projectV1";
import { useCurrentProjectV1 } from "@/store";
import { isValidRolloutName } from "@/types";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { RolloutSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { isValidIssueName } from "@/utils";
import { usePlanContext } from "../context";
import {
  refreshPlan,
  refreshPlanCheckRuns,
  refreshRollout,
  refreshIssue,
  refreshTaskRuns,
} from "./utils";

type ResourceType = "plan" | "planCheckRuns" | "issue" | "rollout" | "taskRuns";

// Progressive polling configuration for the resource poller
// This configuration implements an exponential backoff strategy to reduce server load
// while maintaining responsiveness for active users
//
// - min: 2000ms (2s) - Initial polling interval when starting or after user interaction
// - max: 30000ms (30s) - Maximum polling interval to prevent excessive delays
// - growth: 2 - Growth factor for exponential backoff (2x means: 2s → 4s → 8s → 16s → 30s)
// - jitter: ±3000ms (±3s) - Random variation added to prevent thundering herd problem
//                          where multiple clients poll simultaneously
//
// Example progression with growth=2:
// Poll 1: 2s (initial)
// Poll 2: 4s (2s × 2)
// Poll 3: 8s (4s × 2)
// Poll 4: 16s (8s × 2)
// Poll 5+: 30s (capped at max, would be 32s but limited by max)
//
// The actual interval will vary by ±3s due to jitter, so Poll 3 might be anywhere
// from 5s to 11s (8s ± 3s), helping distribute server load
const POLLER_INTERVAL = { min: 2000, max: 30000, growth: 2, jitter: 3000 };

const KEY = Symbol(
  `bb.plan.poller.${uuidv4()}`
) as InjectionKey<ResourcePollerContext>;

export const provideResourcePoller = () => {
  const route = useRoute();
  const { project } = useCurrentProjectV1();
  const { isCreating, plan, planCheckRuns, taskRuns, issue, rollout, events } =
    usePlanContext();

  // Track refreshing state
  const isRefreshing = ref(false);

  // Track last refresh time
  const lastRefreshTime = ref(0);

  const resourcesFromRoute = computed<ResourceType[]>(() => {
    if (isCreating.value) {
      return [];
    }

    const routeName = route.name as string;
    // Plan-specific pages
    if (
      includes(
        [
          PROJECT_V1_ROUTE_PLAN_DETAIL,
          PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
          PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
        ],
        routeName
      )
    ) {
      return ["plan"];
    }

    // Issue-specific pages
    if (includes([PROJECT_V1_ROUTE_ISSUE_DETAIL_V1], routeName)) {
      return ["issue"];
    }

    // Rollout-specific pages
    if (
      includes(
        [
          PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
          PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
          PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
        ],
        routeName
      )
    ) {
      return ["rollout", "taskRuns"];
    }

    // Default to polling all resources
    return ["plan", "planCheckRuns", "issue", "rollout", "taskRuns"];
  });

  const resourcesToPolled = computed<ResourceType[]>(() => {
    return uniq([...resourcesFromRoute.value]);
  });

  // Create refresh functions for each resource
  const refreshPlanOnly = async () => {
    if (!plan.value) return;
    await refreshPlan(plan);
  };

  const refreshPlanCheckRunsOnly = async () => {
    if (!plan.value || !planCheckRuns) return;
    await refreshPlanCheckRuns(plan.value, project.value, planCheckRuns);
  };

  const refreshIssueOnly = async () => {
    if (!issue?.value) return;
    await refreshIssue(issue as any);
  };

  const refreshRolloutOnly = async () => {
    if (!plan.value?.rollout || !rollout) return;
    await refreshRollout(plan.value.rollout, project.value, rollout);
  };

  const refreshTaskRunsOnly = async () => {
    if (!rollout?.value || !taskRuns) return;
    await refreshTaskRuns(rollout.value, project.value, taskRuns);
  };

  const refreshResources = async (
    resources: ResourceType[] = resourcesToPolled.value,
    force?: boolean
  ) => {
    if ((resources.length === 0 || isRefreshing.value) && !force) return;

    isRefreshing.value = true;
    const refreshPromises = [];

    try {
      if (resources.includes("plan") && plan.value) {
        refreshPromises.push(refreshPlan(plan));
      }
      if (resources.includes("planCheckRuns") && plan.value && planCheckRuns) {
        refreshPromises.push(
          refreshPlanCheckRuns(plan.value, project.value, planCheckRuns)
        );
      }
      if (resources.includes("issue") && issue?.value) {
        refreshPromises.push(refreshIssue(issue as any));
      }
      if (resources.includes("rollout") && plan.value?.rollout && rollout) {
        refreshPromises.push(
          refreshRollout(plan.value.rollout, project.value, rollout)
        );
      }
      if (resources.includes("taskRuns") && rollout?.value && taskRuns) {
        refreshPromises.push(
          refreshTaskRuns(rollout.value, project.value, taskRuns)
        );
      }

      await Promise.all(refreshPromises);

      // Update timestamp after successful refresh
      lastRefreshTime.value = Date.now();

      // Emit event after successful refresh
      events.emit("resource-refresh-completed", {
        resources: resources,
        isManual: false,
      });
    } finally {
      isRefreshing.value = false;
    }

    // If force is true, restart the poller to ensure it continues polling.
    if (force) {
      resourcePoller.restart();
    }
  };

  // Create a single poller for all resources
  const resourcePoller = useProgressivePoll(refreshResources, {
    interval: POLLER_INTERVAL,
  });

  // Track if we've done initial refresh to avoid duplicate calls
  let hasInitialRefresh = false;
  let isPollerRunning = false;

  // Function to restart the poller (resets progressive intervals)
  const restartPoller = () => {
    const shouldPoll = !isCreating.value && resourcesToPolled.value.length > 0;

    if (!shouldPoll) return;

    // Stop the poller first
    resourcePoller.stop();
    isPollerRunning = false;
    hasInitialRefresh = false;

    // Restart the poller
    resourcePoller.start();
    isPollerRunning = true;
  };

  // Watch for route changes and restart poller only when resources actually change
  watch(
    () => resourcesToPolled.value,
    (newResources, oldResources) => {
      // Only restart if the resources to be polled have actually changed
      if (oldResources && newResources.length > 0) {
        // Create sorted arrays to compare
        const newSorted = [...newResources].sort();
        const oldSorted = [...oldResources].sort();

        const resourcesChanged =
          newSorted.length !== oldSorted.length ||
          newSorted.some((resource, index) => resource !== oldSorted[index]);

        if (resourcesChanged) {
          restartPoller();
        }
      }
    },
    { deep: true }
  );

  // Watch for plan issue/rollout changes on plan pages
  // This ensures we fetch issue/rollout once when they are created
  watch(
    () => ({
      issue: plan.value?.issue,
      rollout: plan.value?.rollout,
    }),
    async (newValues, oldValues) => {
      const routeName = route.name as string;
      // Only react to changes on plan-specific pages
      if (
        includes(
          [
            PROJECT_V1_ROUTE_PLAN_DETAIL,
            PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
            PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
          ],
          routeName
        )
      ) {
        // Check if issue or rollout was added (changed from empty to non-empty)
        const issueAdded = !oldValues?.issue && newValues.issue;
        const rolloutAdded = !oldValues?.rollout && newValues.rollout;

        if (issueAdded && isValidIssueName(newValues.issue)) {
          issue.value = create(IssueSchema, {
            name: newValues.issue,
          });
          await refreshIssueOnly();
        }
        if (rolloutAdded && isValidRolloutName(newValues.rollout)) {
          rollout.value = create(RolloutSchema, {
            name: newValues.rollout,
          });
          await refreshRolloutOnly();
        }
        if (issueAdded || rolloutAdded) {
          // Emit status changed to refresh the UI
          events.emit("status-changed", { eager: true });
        }
      }
    }
  );

  const refreshAll = async (isManual = false) => {
    const activeResources = resourcesToPolled.value;
    if (activeResources.length === 0 || isRefreshing.value) return;

    isRefreshing.value = true;
    const refreshPromises = [];

    try {
      if (activeResources.includes("plan"))
        refreshPromises.push(refreshPlanOnly());
      if (activeResources.includes("planCheckRuns"))
        refreshPromises.push(refreshPlanCheckRunsOnly());
      if (activeResources.includes("issue"))
        refreshPromises.push(refreshIssueOnly());
      if (activeResources.includes("rollout"))
        refreshPromises.push(refreshRolloutOnly());
      if (activeResources.includes("taskRuns"))
        refreshPromises.push(refreshTaskRunsOnly());

      await Promise.all(refreshPromises);

      // Update timestamp after successful refresh
      lastRefreshTime.value = Date.now();

      // Emit event after successful refresh
      events.emit("resource-refresh-completed", {
        resources: activeResources,
        isManual,
      });
    } finally {
      isRefreshing.value = false;
    }
  };

  // Set up event listeners
  events.on("status-changed", async ({ eager }) => {
    if (eager) {
      await refreshAll();
    }
  });

  events.on("perform-issue-review-action", async () => {
    await Promise.all([refreshIssueOnly()]);
    events.emit("status-changed", { eager: true });
  });

  events.on("perform-issue-status-action", async () => {
    await refreshIssueOnly();
    events.emit("status-changed", { eager: true });
  });

  // Watch for resource changes and start/stop poller accordingly
  watchEffect(() => {
    const activeResources = resourcesToPolled.value;
    const shouldPoll = !isCreating.value && activeResources.length > 0;

    if (shouldPoll) {
      if (!isPollerRunning) {
        resourcePoller.start();
        isPollerRunning = true;
      }

      // Do initial refresh only once when polling starts
      if (!hasInitialRefresh) {
        hasInitialRefresh = true;
        // Small delay to avoid race conditions with component initialization
        setTimeout(async () => {
          await refreshAll();
        }, 100);
      }
    } else {
      // Stop the poller when creating or no resources to poll
      if (isPollerRunning) {
        resourcePoller.stop();
        isPollerRunning = false;
      }
    }
  });

  const poller = {
    refreshResources,
    isRefreshing,
    lastRefreshTime,
  };

  provide(KEY, poller);
  return poller;
};

type ResourcePollerContext = ReturnType<typeof provideResourcePoller>;

export const useResourcePoller = () => {
  const context = inject(KEY);
  if (!context) {
    throw new Error(
      "useResourcePoller must be called within a component that provides PollerContext"
    );
  }
  return context;
};
