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
  PROJECT_V1_ROUTE_PLAN_DETAIL_CHECK_RUNS,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
  PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
} from "@/router/dashboard/projectV1";
import { useCurrentProjectV1 } from "@/store";
import { usePlanContext } from "../context";
import {
  refreshPlan,
  refreshPlanCheckRuns,
  refreshRollout,
  refreshIssue,
  refreshIssueComments,
} from "./utils";

export type ResourceType =
  | "plan"
  | "planCheckRuns"
  | "issue"
  | "issueComments"
  | "rollout";

// Progressive polling configuration:
// - min: 2s - Initial polling interval
// - max: 60s - Maximum polling interval after progressive growth
// - growth: 1.5x - Each interval is 1.5x the previous (2s → 3s → 4.5s → 6.75s → ...)
// - jitter: ±3s - Random variation to prevent thundering herd
const POLLER_INTERVAL = { min: 2000, max: 60000, growth: 1.5, jitter: 3000 };

const KEY = Symbol(
  `bb.plan.poller.${uuidv4()}`
) as InjectionKey<ResourcePollerContext>;

export const provideResourcePoller = () => {
  const route = useRoute();
  const { project } = useCurrentProjectV1();
  const { isCreating, plan, planCheckRuns, issue, rollout, events } =
    usePlanContext();

  // Track refreshing state
  const isRefreshing = ref(false);

  const enhancedPollingResources = ref<ResourceType[]>([]);

  const requestEnhancedPolling = (
    resources: ResourceType[],
    once?: boolean
  ) => {
    if (once) {
      for (const resource of resources) {
        if (resource === "plan") {
          refreshPlanOnly();
        } else if (resource === "planCheckRuns") {
          refreshPlanCheckRunsOnly();
        } else if (resource === "issue") {
          refreshIssueOnly();
        } else if (resource === "issueComments") {
          refreshIssueCommentsOnly();
        } else if (resource === "rollout") {
          refreshRolloutOnly();
        }
      }
      restartActivePollers();
    } else {
      enhancedPollingResources.value = resources;
    }
  };

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

    if (includes([PROJECT_V1_ROUTE_PLAN_DETAIL_CHECK_RUNS], routeName)) {
      return ["planCheckRuns"];
    }

    // Issue-specific pages
    if (includes([PROJECT_V1_ROUTE_ISSUE_DETAIL_V1], routeName)) {
      return ["issue", "issueComments"];
    }

    // Rollout-specific pages
    if (
      includes(
        [
          PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
          PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
        ],
        routeName
      )
    ) {
      return ["rollout"];
    }

    // Default to polling all resources
    return ["plan", "issue", "issueComments", "rollout"];
  });

  const resourcesToPolled = computed<ResourceType[]>(() => {
    return uniq([
      ...enhancedPollingResources.value,
      ...resourcesFromRoute.value,
    ]);
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

  const refreshIssueCommentsOnly = async () => {
    if (!issue?.value) return;
    await refreshIssueComments(issue.value);
  };

  const refreshRolloutOnly = async () => {
    if (!plan.value?.rollout || !rollout) return;
    await refreshRollout(plan.value.rollout, project.value, rollout);
  };

  // Track which resource is being refreshed for the event
  const resourceBeingRefreshed = ref<ResourceType | null>(null);

  // Wrap refresh functions to manage isRefreshing state for auto polling
  const wrapWithRefreshingState = (
    refreshFn: () => Promise<void>,
    resource: ResourceType
  ) => {
    return async () => {
      isRefreshing.value = true;
      resourceBeingRefreshed.value = resource;
      try {
        await refreshFn();
        // Emit event after successful refresh
        events.emit("resource-refresh-completed", {
          resources: [resource],
          isManual: false,
        });
      } finally {
        isRefreshing.value = false;
        resourceBeingRefreshed.value = null;
      }
    };
  };

  // Create pollers for each resource
  const planPoller = useProgressivePoll(
    wrapWithRefreshingState(refreshPlanOnly, "plan"),
    { interval: POLLER_INTERVAL }
  );

  const planCheckRunsPoller = useProgressivePoll(
    wrapWithRefreshingState(refreshPlanCheckRunsOnly, "planCheckRuns"),
    { interval: POLLER_INTERVAL }
  );

  const issuePoller = useProgressivePoll(
    wrapWithRefreshingState(refreshIssueOnly, "issue"),
    { interval: POLLER_INTERVAL }
  );

  const issueCommentsPoller = useProgressivePoll(
    wrapWithRefreshingState(refreshIssueCommentsOnly, "issueComments"),
    { interval: POLLER_INTERVAL }
  );

  const rolloutPoller = useProgressivePoll(
    wrapWithRefreshingState(refreshRolloutOnly, "rollout"),
    { interval: POLLER_INTERVAL }
  );

  // Track if we've done initial refresh to avoid duplicate calls
  let hasInitialRefresh = false;
  // Track which pollers are currently running to avoid unnecessary restarts
  const pollerStates = {
    plan: false,
    planCheckRuns: false,
    issue: false,
    issueComments: false,
    rollout: false,
  };

  // Function to restart all active pollers (resets progressive intervals)
  const restartActivePollers = () => {
    const activeResources = resourcesToPolled.value;
    const shouldPoll = !isCreating.value;

    if (!shouldPoll) return;

    // Stop all pollers first
    planPoller.stop();
    planCheckRunsPoller.stop();
    issuePoller.stop();
    issueCommentsPoller.stop();
    rolloutPoller.stop();

    // Reset poller states and initial refresh flag
    pollerStates.plan = false;
    pollerStates.planCheckRuns = false;
    pollerStates.issue = false;
    pollerStates.issueComments = false;
    pollerStates.rollout = false;
    hasInitialRefresh = false;

    // Restart active ones (this will reset the progressive intervals)
    if (activeResources.includes("plan")) {
      planPoller.start();
      pollerStates.plan = true;
    }
    if (activeResources.includes("planCheckRuns")) {
      planCheckRunsPoller.start();
      pollerStates.planCheckRuns = true;
    }
    if (activeResources.includes("issue")) {
      issuePoller.start();
      pollerStates.issue = true;
    }
    if (activeResources.includes("issueComments")) {
      issueCommentsPoller.start();
      pollerStates.issueComments = true;
    }
    if (activeResources.includes("rollout")) {
      rolloutPoller.start();
      pollerStates.rollout = true;
    }
  };

  // Watch for route changes and restart pollers only when resources actually change
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
          restartActivePollers();
        }
      }
    },
    { deep: true }
  );

  const refreshAll = async (isManual = false) => {
    const activeResources = resourcesToPolled.value;
    const refreshPromises = [];

    if (activeResources.includes("plan"))
      refreshPromises.push(refreshPlanOnly());
    if (activeResources.includes("planCheckRuns"))
      refreshPromises.push(refreshPlanCheckRunsOnly());
    if (activeResources.includes("issue"))
      refreshPromises.push(refreshIssueOnly());
    if (activeResources.includes("issueComments"))
      refreshPromises.push(refreshIssueCommentsOnly());
    if (activeResources.includes("rollout"))
      refreshPromises.push(refreshRolloutOnly());

    isRefreshing.value = true;
    try {
      await Promise.all(refreshPromises);
      // Emit event after successful refresh
      events.emit("resource-refresh-completed", {
        resources: activeResources,
        isManual,
      });
    } finally {
      isRefreshing.value = false;
    }
  };

  const refreshAllManual = async () => {
    if (isRefreshing.value) return;
    await refreshAll(true);
  };

  // Set up event listeners
  events.on("status-changed", async ({ eager }) => {
    if (eager) {
      await refreshAll();
    }
  });

  events.on("perform-issue-review-action", async () => {
    await Promise.all([refreshIssueOnly(), refreshIssueCommentsOnly()]);
    events.emit("status-changed", { eager: true });
  });

  events.on("perform-issue-status-action", async () => {
    await refreshIssueOnly();
    events.emit("status-changed", { eager: true });
  });

  // Watch for resource changes and start/stop pollers accordingly
  watchEffect(() => {
    const activeResources = resourcesToPolled.value;
    const shouldPoll = !isCreating.value;

    if (shouldPoll) {
      if (activeResources.includes("plan") && !pollerStates.plan) {
        planPoller.start();
        pollerStates.plan = true;
      } else if (!activeResources.includes("plan") && pollerStates.plan) {
        planPoller.stop();
        pollerStates.plan = false;
      }

      if (
        activeResources.includes("planCheckRuns") &&
        !pollerStates.planCheckRuns
      ) {
        planCheckRunsPoller.start();
        pollerStates.planCheckRuns = true;
      } else if (
        !activeResources.includes("planCheckRuns") &&
        pollerStates.planCheckRuns
      ) {
        planCheckRunsPoller.stop();
        pollerStates.planCheckRuns = false;
      }

      if (activeResources.includes("issue") && !pollerStates.issue) {
        issuePoller.start();
        pollerStates.issue = true;
      } else if (!activeResources.includes("issue") && pollerStates.issue) {
        issuePoller.stop();
        pollerStates.issue = false;
      }

      if (
        activeResources.includes("issueComments") &&
        !pollerStates.issueComments
      ) {
        issueCommentsPoller.start();
        pollerStates.issueComments = true;
      } else if (
        !activeResources.includes("issueComments") &&
        pollerStates.issueComments
      ) {
        issueCommentsPoller.stop();
        pollerStates.issueComments = false;
      }

      if (activeResources.includes("rollout") && !pollerStates.rollout) {
        rolloutPoller.start();
        pollerStates.rollout = true;
      } else if (!activeResources.includes("rollout") && pollerStates.rollout) {
        rolloutPoller.stop();
        pollerStates.rollout = false;
      }

      // Do initial refresh only once when polling starts
      if (!hasInitialRefresh && activeResources.length > 0) {
        hasInitialRefresh = true;
        // Small delay to avoid race conditions with component initialization
        setTimeout(() => refreshAll(), 100);
      }
    } else {
      // Stop all pollers when creating
      planPoller.stop();
      planCheckRunsPoller.stop();
      issuePoller.stop();
      issueCommentsPoller.stop();
      rolloutPoller.stop();
      // Reset poller states
      pollerStates.plan = false;
      pollerStates.planCheckRuns = false;
      pollerStates.issue = false;
      pollerStates.issueComments = false;
      pollerStates.rollout = false;
    }
  });

  const poller = {
    refreshAllManual,
    requestEnhancedPolling,
    restartActivePollers,
    isRefreshing: computed(() => isRefreshing.value),
    activeResources: resourcesToPolled,
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
