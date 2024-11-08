import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide, watchEffect } from "vue";
import { useRoute } from "vue-router";
import { useProjectV1Store, useRolloutStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import type { ComposedProject, ComposedRollout } from "@/types";
import { unknownProject, unknownRollout } from "@/types";

export type RolloutDetailContext = {
  rollout: Ref<ComposedRollout>;
  project: ComputedRef<ComposedProject>;
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

  const project = computed(() => {
    const projectId = route.params.projectId as string;
    if (!projectId) {
      return unknownProject();
    }
    return projectV1Store.getProjectByName(`${projectNamePrefix}${projectId}`);
  });

  watchEffect(async () => {
    await rolloutStore.fetchRolloutByName(rolloutName);
  });

  const rollout = computed(() => {
    return rolloutStore.getRolloutByName(rolloutName) ?? unknownRollout();
  });

  const context: RolloutDetailContext = {
    rollout,
    project,
  };

  provide(KEY, context);

  return context;
};
