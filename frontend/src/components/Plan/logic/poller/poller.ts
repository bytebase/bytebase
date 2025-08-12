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

// Progressive polling configuration.
const POLLER_INTERVAL = { min: 2000, max: 30000, growth: 2, jitter: 3000 };

const KEY = Symbol(
  `bb.plan.poller.${uuidv4()}`
) as InjectionKey<ResourcePollerContext>;

// Resource refresh strategies
interface ResourceRefreshStrategy {
  canRefresh: () => boolean;
  refresh: () => Promise<void>;
  canInitialize?: () => boolean;
  initialize?: () => Promise<void>;
}

export const provideResourcePoller = () => {
  const route = useRoute();
  const { project } = useCurrentProjectV1();
  const { isCreating, plan, planCheckRuns, taskRuns, issue, rollout, events } =
    usePlanContext();

  // Consolidated state management
  const pollerState = ref({
    isRefreshing: false,
    lastRefreshTime: 0,
    hasInitialRefresh: false,
    hasInitialResourceSetup: false,
    isPollerRunning: false,
  });

  // Define refresh strategies for each resource
  const resourceStrategies: Record<ResourceType, ResourceRefreshStrategy> = {
    plan: {
      canRefresh: () => !!plan.value,
      refresh: () => refreshPlan(plan),
    },
    planCheckRuns: {
      canRefresh: () => !!plan.value && !!planCheckRuns,
      refresh: () =>
        refreshPlanCheckRuns(plan.value, project.value, planCheckRuns),
    },
    issue: {
      canRefresh: () => !!issue?.value,
      refresh: () => refreshIssue(issue as any),
      canInitialize: () =>
        !!(plan.value?.issue && isValidIssueName(plan.value.issue)),
      initialize: async () => {
        if (!plan.value?.issue) return;
        issue.value = create(IssueSchema, { name: plan.value.issue });
        await resourceStrategies.issue.refresh();
      },
    },
    rollout: {
      canRefresh: () => !!plan.value?.rollout && !!rollout,
      refresh: () => refreshRollout(plan.value.rollout, project.value, rollout),
      canInitialize: () =>
        !!plan.value?.rollout && isValidRolloutName(plan.value.rollout),
      initialize: async () => {
        if (!plan.value?.rollout) return;
        rollout.value = create(RolloutSchema, { name: plan.value.rollout });
        await resourceStrategies.rollout.refresh();
      },
    },
    taskRuns: {
      canRefresh: () => !!rollout?.value && !!taskRuns,
      refresh: () => refreshTaskRuns(rollout.value!, project.value, taskRuns),
    },
  };

  const resourcesFromRoute = computed<ResourceType[]>(() => {
    if (isCreating.value) return [];

    const routeName = route.name as string;
    const planRoutes = [
      PROJECT_V1_ROUTE_PLAN_DETAIL,
      PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
      PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
    ];
    const rolloutRoutes = [
      PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
      PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
      PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
    ];

    if (includes(planRoutes, routeName)) return ["plan"];
    if (includes([PROJECT_V1_ROUTE_ISSUE_DETAIL_V1], routeName))
      return ["issue"];
    if (includes(rolloutRoutes, routeName)) return ["rollout", "taskRuns"];

    return ["plan", "planCheckRuns", "issue", "rollout", "taskRuns"];
  });

  const resourcesToPolled = computed(() => uniq([...resourcesFromRoute.value]));

  // Initialize existing resources that are already on the plan
  const initializeExistingResources = async () => {
    if (
      pollerState.value.hasInitialResourceSetup ||
      isCreating.value ||
      !plan.value
    ) {
      return;
    }

    pollerState.value.hasInitialResourceSetup = true;
    const initPromises: Promise<void>[] = [];

    // Initialize issue if needed
    if (
      resourceStrategies.issue.canInitialize?.() &&
      resourceStrategies.issue.initialize
    ) {
      initPromises.push(resourceStrategies.issue.initialize());
    }

    // Initialize rollout and taskRuns with proper sequencing
    if (
      resourceStrategies.rollout.canInitialize?.() &&
      resourceStrategies.rollout.initialize
    ) {
      initPromises.push(
        resourceStrategies.rollout.initialize().then(async () => {
          if (resourceStrategies.taskRuns.canRefresh()) {
            await resourceStrategies.taskRuns.refresh();
          }
        })
      );
    }

    if (initPromises.length > 0) {
      await Promise.all(initPromises);
      events.emit("status-changed", { eager: true });
    }
  };

  // Unified refresh function (with backward compatibility)
  const refreshResources = async (
    resources: ResourceType[] = resourcesToPolled.value,
    optionsOrForce: { force?: boolean; isManual?: boolean } | boolean = {}
  ) => {
    // Handle backward compatibility for boolean force parameter
    const options =
      typeof optionsOrForce === "boolean"
        ? { force: optionsOrForce, isManual: true }
        : optionsOrForce;
    const { force = false, isManual = false } = options;

    if ((resources.length === 0 || pollerState.value.isRefreshing) && !force) {
      return;
    }

    pollerState.value.isRefreshing = true;
    const refreshPromises: Promise<void>[] = [];

    try {
      for (const resourceType of resources) {
        const strategy = resourceStrategies[resourceType];
        if (strategy.canRefresh()) {
          refreshPromises.push(strategy.refresh());
        }
      }

      await Promise.all(refreshPromises);

      pollerState.value.lastRefreshTime = Date.now();
      events.emit("resource-refresh-completed", { resources, isManual });
    } finally {
      pollerState.value.isRefreshing = false;
    }

    if (force) {
      resourcePoller.restart();
    }
  };

  // Create the poller
  const resourcePoller = useProgressivePoll(refreshResources, {
    interval: POLLER_INTERVAL,
  });

  // Reset poller state
  const resetPollerState = () => {
    pollerState.value.hasInitialRefresh = false;
    pollerState.value.hasInitialResourceSetup = false;
    pollerState.value.isPollerRunning = false;
  };

  // Restart poller with clean state
  const restartPoller = () => {
    const shouldPoll = !isCreating.value && resourcesToPolled.value.length > 0;
    if (!shouldPoll) return;

    resourcePoller.stop();
    resetPollerState();
    resourcePoller.start();
    pollerState.value.isPollerRunning = true;
  };

  // Watch for route changes and restart poller when resources change
  watch(
    resourcesToPolled,
    (newResources, oldResources) => {
      if (!oldResources || newResources.length === 0) return;

      const newSorted = [...newResources].sort();
      const oldSorted = [...oldResources].sort();
      const resourcesChanged =
        newSorted.length !== oldSorted.length ||
        newSorted.some((resource, index) => resource !== oldSorted[index]);

      if (resourcesChanged) {
        restartPoller();
      }
    },
    { deep: true }
  );

  // Watch for plan issue/rollout changes on plan pages
  watch(
    () => ({ issue: plan.value?.issue, rollout: plan.value?.rollout }),
    async (newValues, oldValues) => {
      const routeName = route.name as string;
      const isPlanRoute = includes(
        [
          PROJECT_V1_ROUTE_PLAN_DETAIL,
          PROJECT_V1_ROUTE_PLAN_DETAIL_SPECS,
          PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
        ],
        routeName
      );

      if (!isPlanRoute) return;

      const issueAdded = !oldValues?.issue && newValues.issue;
      const rolloutAdded = !oldValues?.rollout && newValues.rollout;

      if (issueAdded && resourceStrategies.issue.initialize) {
        await resourceStrategies.issue.initialize();
      }
      if (rolloutAdded && resourceStrategies.rollout.initialize) {
        await resourceStrategies.rollout.initialize();
      }
      if (issueAdded || rolloutAdded) {
        events.emit("status-changed", { eager: true });
      }
    }
  );

  // Event listeners
  events.on("status-changed", async ({ eager }) => {
    if (eager) {
      await refreshResources(resourcesToPolled.value, { isManual: false });
    }
  });

  events.on("perform-issue-review-action", async () => {
    await refreshResources(["issue"], { isManual: true });
    events.emit("status-changed", { eager: true });
  });

  events.on("perform-issue-status-action", async () => {
    await refreshResources(["issue"], { isManual: true });
    events.emit("status-changed", { eager: true });
  });

  // Main poller lifecycle management
  watchEffect(async () => {
    const activeResources = resourcesToPolled.value;
    const shouldPoll = !isCreating.value && activeResources.length > 0;

    if (shouldPoll) {
      await initializeExistingResources();

      if (!pollerState.value.isPollerRunning) {
        resourcePoller.start();
        pollerState.value.isPollerRunning = true;
      }

      if (!pollerState.value.hasInitialRefresh) {
        pollerState.value.hasInitialRefresh = true;
        setTimeout(async () => {
          await refreshResources(activeResources, { isManual: false });
        }, 100);
      }
    } else if (pollerState.value.isPollerRunning) {
      resourcePoller.stop();
      pollerState.value.isPollerRunning = false;
    }
  });

  const poller = {
    refreshResources: (
      resources?: ResourceType[],
      optionsOrForce?: { force?: boolean; isManual?: boolean } | boolean
    ) => refreshResources(resources, optionsOrForce),
    isRefreshing: computed(() => pollerState.value.isRefreshing),
    lastRefreshTime: computed(() => pollerState.value.lastRefreshTime),
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
