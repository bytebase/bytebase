import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import type { InjectionKey } from "vue";
import { computed, inject, provide, ref, onUnmounted } from "vue";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import { useCurrentProjectV1 } from "@/store";
import {
  CreateRolloutRequestSchema,
  RolloutSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import type { Rollout } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContextWithRollout } from "../../logic";

export const KEY = Symbol(
  "bb.plan.rollout-view"
) as InjectionKey<RolloutViewContext>;

export const useRolloutViewContext = () => {
  return inject(KEY)!;
};

export const provideRolloutViewContext = () => {
  const { events, rollout } = usePlanContextWithRollout();
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
      // Validate that rollout has required fields before making the request
      if (!rollout.value?.plan || !project.value?.name) {
        rolloutPreview.value = create(RolloutSchema, {});
        return;
      }

      const rolloutPreviewNew = await rolloutServiceClientConnect.createRollout(
        create(CreateRolloutRequestSchema, {
          parent: project.value.name,
          rollout: rollout.value,
          validateOnly: true,
        }),
        {
          contextValues: createContextValues().set(silentContextKey, true),
        }
      );
      rolloutPreview.value = rolloutPreviewNew;
    } catch (error) {
      // Handle preview errors gracefully
      console.error("Failed to fetch rollout preview:", error);
      rolloutPreview.value = create(RolloutSchema, {});
    } finally {
      ready.value = true;
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
