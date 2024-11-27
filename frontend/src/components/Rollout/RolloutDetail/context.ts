import Emittery from "emittery";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { useRoute } from "vue-router";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { useIssueV1Store, useProjectV1Store, useRolloutStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { ComposedIssue, ComposedProject, ComposedRollout } from "@/types";
import { unknownProject, unknownRollout } from "@/types";
import type { Task } from "@/types/proto/v1/rollout_service";
import { flattenTaskV1List } from "@/utils";

type Events = {
  "task-status-action": undefined;
};

export type EventsEmmiter = Emittery<Events>;

export type RolloutDetailContext = {
  rollout: Ref<ComposedRollout>;
  issue: Ref<ComposedIssue | undefined>;

  project: ComputedRef<ComposedProject>;
  tasks: ComputedRef<Task[]>;

  // The events emmiter.
  emmiter: EventsEmmiter;
};

export const KEY = Symbol(
  "bb.rollout.detail"
) as InjectionKey<RolloutDetailContext>;

export const useRolloutDetailContext = () => {
  return inject(KEY)!;
};

export const provideRolloutDetailContext = (rolloutName: string) => {
  const route = useRoute();
  const projectV1Store = useProjectV1Store();
  const rolloutStore = useRolloutStore();
  const issueStore = useIssueV1Store();

  const rollout = ref<ComposedRollout>(unknownRollout());
  const issue = ref<ComposedIssue | undefined>(undefined);

  const project = computed(() => {
    const projectId = route.params.projectId as string;
    if (!projectId) {
      return unknownProject();
    }
    return projectV1Store.getProjectByName(`${projectNamePrefix}${projectId}`);
  });

  const tasks = computed(() => flattenTaskV1List(rollout.value));

  const emmiter: EventsEmmiter = new Emittery<Events>();
  // When any task status action is triggered, we need to refresh the rollout.
  emmiter.on("task-status-action", () => {
    refreshRolloutContext();
    poller.restart();
  });

  const context: RolloutDetailContext = {
    project,
    rollout,
    issue,
    tasks,
    emmiter,
  };

  provide(KEY, context);

  const refreshRolloutContext = async () => {
    rollout.value = await rolloutStore.fetchRolloutByName(rolloutName);
    if (rollout.value.issue) {
      issue.value = await issueStore.fetchIssueByName(rollout.value.issue, {
        // Don't need to fetch the plan and rollout.
        withPlan: false,
        withRollout: false,
      });
    }
  };

  refreshRolloutContext();

  // Poll the rollout status.
  const poller = useProgressivePoll(refreshRolloutContext, {
    interval: {
      min: 500,
      max: 10000,
      growth: 2,
      jitter: 500,
    },
  });
  poller.start();

  return context;
};
