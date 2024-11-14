import Emittery from "emittery";
import type { ClientError } from "nice-grpc-common";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide, watchEffect } from "vue";
import { useRoute } from "vue-router";
import { pushNotification, useProjectV1Store, useRolloutStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { ComposedProject, ComposedRollout } from "@/types";
import { unknownProject, unknownRollout } from "@/types";

type Events = {
  "task-status-action": undefined;
};

export type EventsEmmiter = Emittery<Events>;

export type RolloutDetailContext = {
  rollout: Ref<ComposedRollout>;
  project: ComputedRef<ComposedProject>;

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
  const emmiter: EventsEmmiter = new Emittery<Events>();

  const project = computed(() => {
    const projectId = route.params.projectId as string;
    if (!projectId) {
      return unknownProject();
    }
    return projectV1Store.getProjectByName(`${projectNamePrefix}${projectId}`);
  });

  watchEffect(async () => {
    try {
      await rolloutStore.fetchRolloutByName(rolloutName);
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Failed to get rollout",
        description: (error as ClientError).details,
        manualHide: true,
      });
    }
  });

  const rollout = computed(() => {
    return rolloutStore.getRolloutByName(rolloutName) ?? unknownRollout();
  });

  const context: RolloutDetailContext = {
    rollout,
    project,
    emmiter,
  };

  provide(KEY, context);

  return context;
};
