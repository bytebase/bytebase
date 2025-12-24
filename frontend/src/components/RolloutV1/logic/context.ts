import { create } from "@bufbuild/protobuf";
import type { InjectionKey } from "vue";
import { computed, inject, nextTick, onUnmounted, provide, ref } from "vue";
import {
  generateRolloutPreview,
  usePlanContextWithRollout,
} from "@/components/Plan/logic";
import { useCurrentProjectV1 } from "@/store";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { RolloutSchema } from "@/types/proto-es/v1/rollout_service_pb";

export const KEY = Symbol(
  "bb.plan.rollout-view"
) as InjectionKey<RolloutViewContext>;

export const useRolloutViewContext = () => {
  return inject(KEY)!;
};

export const provideRolloutViewContext = () => {
  const { events, rollout, plan } = usePlanContextWithRollout();
  const { project } = useCurrentProjectV1();

  const ready = ref(false);
  const rolloutPreview = ref<Rollout>(create(RolloutSchema, {}));

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
    try {
      // Validate that plan and project are available
      if (!plan.value?.name || !project.value?.name) {
        rolloutPreview.value = create(RolloutSchema, {});
        return;
      }

      // Generate rollout preview locally instead of calling backend
      const preview = await generateRolloutPreview(
        plan.value,
        project.value.name
      );
      rolloutPreview.value = preview;
    } catch (error) {
      console.error("Failed to generate rollout preview:", error);
      rolloutPreview.value = create(RolloutSchema, {});
    } finally {
      nextTick(() => {
        ready.value = true;
      });
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

  const context = {
    ready,
    rollout,
    mergedStages,
  };

  provide(KEY, context);
  return context;
};

type RolloutViewContext = ReturnType<typeof provideRolloutViewContext>;
