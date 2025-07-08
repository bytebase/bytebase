<template>
  <div
    class="flex flex-col gap-y-4 max-h-96 overflow-auto border rounded p-6 bg-zinc-50"
  >
    <div v-if="loading" class="flex items-center justify-center py-8">
      <BBSpin />
    </div>
    <div v-else-if="error" class="text-error">
      {{ error }}
    </div>
    <StagesView
      v-else-if="previewRollout"
      :rollout="previewRollout"
      :merged-stages="previewStages"
      :readonly="true"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { computed, ref, watch, toRef } from "vue";
import { BBSpin } from "@/bbkit";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { useCurrentProjectV1 } from "@/store";
import type { Plan } from "@/types/proto-es/v1/plan_service_pb";
import {
  PreviewRolloutRequestSchema,
  type Rollout,
  type Stage,
} from "@/types/proto-es/v1/rollout_service_pb";
import StagesView from "./RolloutView/StagesView.vue";

const props = defineProps<{
  plan: Plan;
}>();

const emit = defineEmits<{
  (
    event: "update:validation",
    validation: {
      isValid: boolean;
      error?: string;
    }
  ): void;
}>();

const { project } = useCurrentProjectV1();
const loading = ref(false);
const error = ref<string>("");
const previewRollout = ref<Rollout | undefined>();

const previewStages = computed((): Stage[] => {
  if (!previewRollout.value) return [];
  return previewRollout.value.stages || [];
});

const totalTasks = computed(() => {
  return previewStages.value.reduce(
    (sum, stage) => sum + stage.tasks.length,
    0
  );
});

// Emit validation state whenever it changes
watch(
  [loading, error, previewStages, totalTasks],
  () => {
    if (loading.value) {
      // Don't emit validation while loading
      return;
    }

    if (error.value) {
      emit("update:validation", {
        isValid: false,
        error: error.value,
      });
    } else if (previewStages.value.length === 0) {
      emit("update:validation", {
        isValid: false,
        error: "No stages in preview rollout",
      });
    } else if (totalTasks.value === 0) {
      emit("update:validation", {
        isValid: false,
        error: "No tasks in preview rollout",
      });
    } else {
      emit("update:validation", {
        isValid: true,
      });
    }
  },
  { immediate: true }
);

const fetchPreviewRollout = async () => {
  loading.value = true;
  error.value = "";

  try {
    const request = create(PreviewRolloutRequestSchema, {
      project: project.value.name,
      plan: props.plan,
    });

    const response = await rolloutServiceClientConnect.previewRollout(request);
    previewRollout.value = response;
  } catch (err) {
    error.value = String(err);
    previewRollout.value = undefined;
  } finally {
    loading.value = false;
  }
};

// Watch for actual plan changes by watching the plan's name
// This ensures we only fetch when the plan actually changes, not on every render
const planRef = toRef(props, "plan");

watch(
  () => planRef.value?.name,
  (newName, oldName) => {
    if (newName && newName !== oldName) {
      fetchPreviewRollout();
    }
  },
  {
    immediate: true,
  }
);
</script>
