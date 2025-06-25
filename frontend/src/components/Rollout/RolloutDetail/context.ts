import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { computedAsync } from "@vueuse/core";
import Emittery from "emittery";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref } from "vue";
import { useRoute } from "vue-router";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import { useIssueV1Store, useProjectV1Store, useRolloutStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { ComposedIssue, ComposedProject, ComposedRollout } from "@/types";
import { unknownProject, unknownRollout } from "@/types";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  Rollout,
  type Task,
  type Stage,
} from "@/types/proto/v1/rollout_service";
import { flattenTaskV1List } from "@/utils";
import { convertNewRolloutToOld } from "@/utils/v1/rollout-conversions";

type Events = {
  "task-status-action": undefined;
};

export type EventsEmmiter = Emittery<Events>;

export type RolloutDetailContext = {
  rollout: Ref<ComposedRollout>;
  rolloutPreview: Ref<Rollout>;
  issue: Ref<ComposedIssue | undefined>;

  project: Ref<ComposedProject>;
  tasks: ComputedRef<Task[]>;
  mergedStages: ComputedRef<Stage[]>;

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
  const rolloutPreview = ref<Rollout>(Rollout.fromPartial({}));
  const issue = ref<ComposedIssue | undefined>(undefined);

  const project = computedAsync(async () => {
    const projectId = route.params.projectId as string;
    if (!projectId) {
      return unknownProject();
    }
    return await projectV1Store.getOrFetchProjectByName(
      `${projectNamePrefix}${projectId}`
    );
  }, unknownProject());

  const tasks = computed(() => flattenTaskV1List(rollout.value));

  const emmiter: EventsEmmiter = new Emittery<Events>();
  // When any task status action is triggered, we need to refresh the rollout.
  emmiter.on("task-status-action", () => {
    refreshRolloutContext();
    poller.restart();
  });

  const mergedStages = computed(() => {
    // Merge preview stages with created rollout stages.
    return [
      ...rollout.value.stages,
      ...rolloutPreview.value.stages.filter(
        (stage) =>
          !rollout.value.stages.some((s) => s.environment === stage.environment)
      ),
    ];
  });

  const context: RolloutDetailContext = {
    project,
    rollout,
    rolloutPreview,
    issue,
    tasks,
    emmiter,
    mergedStages,
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
    const request = create(CreateRolloutRequestSchema, {
      parent: project.value.name,
      rollout: {
        plan: rollout.value.plan,
      },
      validateOnly: true,
    });
    try {
      const rolloutPreviewNew = await rolloutServiceClientConnect.createRollout(
        request,
        {
          contextValues: createContextValues().set(silentContextKey, true),
        }
      );
      rolloutPreview.value = convertNewRolloutToOld(rolloutPreviewNew);
    } catch {
      rolloutPreview.value = Rollout.fromPartial({});
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
