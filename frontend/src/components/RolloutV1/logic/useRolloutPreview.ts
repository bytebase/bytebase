import type { Ref } from "vue";
import { computed, ref } from "vue";
import { generateRolloutPreview } from "@/components/Plan/logic";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";

export const useRolloutPreview = (
  rollout: Ref<Rollout>,
  plan: Ref<Plan>,
  projectName: string
) => {
  const ready = ref(false);
  const previewStages = ref<Stage[]>([]);

  generateRolloutPreview(plan.value, projectName)
    .then((preview) => {
      previewStages.value = preview.stages;
    })
    .catch(() => {})
    .finally(() => {
      ready.value = true;
    });

  const mergedStages = computed(() => {
    const created = rollout.value.stages;
    const createdEnvs = new Set(created.map((s) => s.environment));
    return [
      ...created,
      ...previewStages.value.filter((s) => !createdEnvs.has(s.environment)),
    ];
  });

  return { ready, mergedStages };
};
