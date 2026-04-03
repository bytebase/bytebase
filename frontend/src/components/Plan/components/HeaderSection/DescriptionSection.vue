<template>
  <div class="flex-1">
    <template v-if="!state.isExpanded">
      <NButton
        v-if="!state.description && allowEdit"
        text
        size="small"
        class="italic opacity-60"
        @click="handleExpand($event)"
      >
        <template #icon>
          <PlusIcon class="w-4 h-4" />
        </template>
        {{ $t("plan.description.placeholder") }}
      </NButton>
      <div v-else class="mt-1">
        <div
          class="relative px-2 py-1 rounded-md border border-transparent transition-all duration-200"
          :class="[
            allowEdit ? 'cursor-pointer hover:border-gray-200' : '',
            !state.showFullDescription ? 'max-h-[4.5rem] overflow-hidden' : '',
          ]"
          @click="allowEdit && handleExpand($event)"
        >
          <MarkdownEditor
            mode="preview"
            class="pointer-events-none"
            :content="state.description"
            :project="project"
          />
          <!-- Fade overlay when truncated -->
          <div
            v-if="!state.showFullDescription && isDescriptionLong"
            class="absolute bottom-0 left-0 right-0 h-6 bg-gradient-to-t from-white to-transparent pointer-events-none"
          />
        </div>
        <button
          v-if="isDescriptionLong"
          class="text-xs text-control-placeholder hover:text-control mt-1 px-2"
          @click.stop="state.showFullDescription = !state.showFullDescription"
        >
          {{ state.showFullDescription ? $t("common.show-less") : $t("common.show-more") }}
        </button>
      </div>
    </template>
    <div v-else class="py-2">
      <div class="flex items-center justify-between">
        <span class="text-base font-medium">{{
          $t("common.description")
        }}</span>
        <div class="flex items-center gap-2">
          <template v-if="!isCreating">
            <NButton
              size="small"
              :disabled="
                state.isUpdating ||
                !allowEdit ||
                state.description === plan.description
              "
              @click="handleSave"
            >
              {{ $t("common.save") }}
            </NButton>
            <NButton size="small" quaternary @click="handleCancel">
              {{ $t("common.cancel") }}
            </NButton>
          </template>
          <NButton v-else size="tiny" @click="handleCancel">
            <template #icon>
              <ChevronUpIcon class="w-4 h-4" />
            </template>
            {{ $t("common.collapse") }}
          </NButton>
        </div>
      </div>
      <MarkdownEditor
        :content="state.description"
        mode="editor"
        :autofocus="state.shouldAutoFocus"
        :project="project"
        :placeholder="$t('plan.description.placeholder')"
        @change="onUpdateValue"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { ChevronUpIcon, PlusIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive, watch } from "vue";
import MarkdownEditor from "@/components/MarkdownEditor";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import { planServiceClientConnect } from "@/connect";
import { useCurrentProjectV1 } from "@/store";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { usePlanContext } from "../../logic";

const { project } = useCurrentProjectV1();
const {
  isCreating,
  plan,
  readonly,
  issue,
  allowEdit: hasPermission,
} = usePlanContext();
const { refreshResources } = useResourcePoller();

const state = reactive({
  isUpdating: false,
  description: plan.value.description,
  isExpanded: false,
  shouldAutoFocus: false,
  justExpanded: false,
  showFullDescription: false,
});

// Description longer than ~3 lines of text
const isDescriptionLong = computed(() => {
  return (
    (state.description?.length ?? 0) > 150 ||
    (state.description?.split("\n").length ?? 0) > 3
  );
});

const allowEdit = computed(() => {
  if (readonly.value) {
    return false;
  }
  if (isCreating.value) {
    return true;
  }
  // Plans with rollout should have readonly description
  if (!issue.value && plan.value.hasRollout) {
    return false;
  }
  return hasPermission.value;
});

const handleExpand = (event: MouseEvent) => {
  if (!allowEdit.value) return;
  event.stopPropagation();
  state.shouldAutoFocus = true;
  state.isExpanded = true;
  state.justExpanded = true;
  // Add a small delay before allowing click outside to work
  setTimeout(() => {
    state.shouldAutoFocus = false;
    state.justExpanded = false;
  }, 100);
};

const handleCancel = () => {
  state.description = plan.value.description;
  state.isExpanded = false;
  if (isCreating.value) {
    plan.value.description = state.description;
  }
};

const handleSave = async () => {
  if (isCreating.value) {
    plan.value.description = state.description;
    state.isExpanded = false;
    return;
  }

  if (!allowEdit.value) return;

  try {
    state.isUpdating = true;
    const planPatch = create(PlanSchema, {
      ...plan.value,
      description: state.description,
    });
    const request = create(UpdatePlanRequestSchema, {
      plan: planPatch,
      updateMask: { paths: ["description"] },
    });
    const response = await planServiceClientConnect.updatePlan(request);
    Object.assign(plan.value, response);
    refreshResources(["plan"], true /** force */);
    state.isExpanded = false;
  } finally {
    state.isUpdating = false;
  }
};

const onUpdateValue = (value: string) => {
  state.description = value;
  if (isCreating.value) {
    plan.value.description = state.description;
  }
};

watch(
  () => plan.value.description,
  (newValue) => {
    if (state.description !== newValue) {
      state.description = newValue;
    }
  },
  { immediate: true }
);
</script>
