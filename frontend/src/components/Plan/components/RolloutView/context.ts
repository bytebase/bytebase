import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import type { ComputedRef, InjectionKey, Ref } from "vue";
import { computed, inject, provide, ref, onUnmounted } from "vue";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import { useCurrentProjectV1 } from "@/store";
import { PreviewRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { Rollout, type Stage } from "@/types/proto/v1/rollout_service";
import {
  convertNewRolloutToOld,
  convertOldPlanToNew,
} from "@/utils/v1/rollout-conversions";
import { usePlanContextWithRollout } from "../../logic";

export type RolloutViewContext = {
  rollout: Ref<Rollout>;
  rolloutPreview: Ref<Rollout | undefined>;
  mergedStages: ComputedRef<Stage[]>;
};

export const KEY = Symbol(
  "bb.plan.rollout-view"
) as InjectionKey<RolloutViewContext>;

export const useRolloutViewContext = () => {
  return inject(KEY)!;
};

export const provideRolloutViewContext = () => {
  const { events, plan, rollout } = usePlanContextWithRollout();
  const { project } = useCurrentProjectV1();

  const rolloutPreview = ref<Rollout>(Rollout.fromPartial({}));

  const mergedStages = computed(() => {
    // Merge preview stages with created rollout stages
    const createdStages = rollout.value.stages;
    const previewStages = rolloutPreview.value.stages;

    // Start with created stages
    const merged = [...createdStages];

    // Add preview stages that don't exist in created stages
    for (const previewStage of previewStages) {
      const existingIndex = merged.findIndex(
        (s) => s.environment === previewStage.environment
      );
      if (existingIndex === -1) {
        merged.push(previewStage);
      }
    }

    return merged;
  });

  const fetchRolloutPreview = async () => {
    const request = create(PreviewRolloutRequestSchema, {
      project: project.value.name,
      plan: convertOldPlanToNew(plan.value),
    });

    try {
      const rolloutPreviewNew =
        await rolloutServiceClientConnect.previewRollout(request, {
          contextValues: createContextValues().set(silentContextKey, true),
        });
      rolloutPreview.value = convertNewRolloutToOld(rolloutPreviewNew);
    } catch (error) {
      // Handle preview errors gracefully
      console.error("Failed to fetch rollout preview:", error);
      rolloutPreview.value = Rollout.fromPartial({});
    } finally {
    }
  };

  // Initial fetch
  fetchRolloutPreview();

  // Listen for resource refresh completion
  const unsubscribe = events.on(
    "resource-refresh-completed",
    async ({ resources }) => {
      // Refresh rollout preview if rollout was refreshed
      if (resources.includes("rollout")) {
        await fetchRolloutPreview();
      }
    }
  );

  // Clean up event listener when component unmounts.
  onUnmounted(() => {
    unsubscribe();
  });

  const context: RolloutViewContext = {
    rollout,
    rolloutPreview,
    mergedStages,
  };

  provide(KEY, context);
  return context;
};
