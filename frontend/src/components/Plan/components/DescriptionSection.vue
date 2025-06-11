<template>
  <div class="">
    <div class="flex items-center justify-between mb-3">
      <h2 class="text-lg font-semibold">{{ $t("common.description") }}</h2>

      <div v-if="!isCreating && allowEdit" class="flex items-center gap-x-2">
        <NButton
          v-if="!state.isEditing"
          size="small"
          @click.prevent="beginEdit"
        >
          {{ $t("common.edit") }}
        </NButton>
        <NButton
          v-if="state.isEditing"
          size="small"
          :disabled="state.description === plan.description"
          :loading="state.isUpdating"
          @click.prevent="saveEdit"
        >
          {{ $t("common.save") }}
        </NButton>
        <NButton
          v-if="state.isEditing"
          size="small"
          quaternary
          @click.prevent="cancelEdit"
        >
          {{ $t("common.cancel") }}
        </NButton>
      </div>
    </div>

    <NInput
      v-if="isCreating || state.isEditing"
      ref="inputRef"
      v-model:value="state.description"
      :placeholder="$t('issue.add-some-description')"
      :autosize="{ minRows: 3, maxRows: 10 }"
      :disabled="state.isUpdating"
      :loading="state.isUpdating"
      style="
        width: 100%;
        --n-placeholder-color: rgb(var(--color-control-placeholder));
      "
      type="textarea"
      @update:value="onDescriptionChange"
    />
    <div
      v-else
      class="min-h-[4rem] max-h-[12rem] whitespace-pre-wrap px-[10px] py-[4.5px] border rounded-md"
    >
      <template v-if="plan.description">
        <iframe
          v-if="plan.description"
          ref="contentPreviewArea"
          :srcdoc="renderedContent"
          class="w-full overflow-hidden"
        />
      </template>
      <span v-else class="text-control-placeholder">
        {{ $t("issue.add-some-description") }}
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInput, NButton } from "naive-ui";
import { computed, nextTick, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRenderMarkdown } from "@/components/MarkdownEditor";
import { planServiceClient } from "@/grpcweb";
import {
  pushNotification,
  useCurrentUserV1,
  extractUserId,
  useCurrentProjectV1,
} from "@/store";
import { Plan } from "@/types/proto/v1/plan_service";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContext } from "../logic";

type LocalState = {
  isEditing: boolean;
  isUpdating: boolean;
  description: string;
};

const { t } = useI18n();
const { project } = useCurrentProjectV1();
const { isCreating, plan } = usePlanContext();
const currentUser = useCurrentUserV1();
const contentPreviewArea = ref<HTMLIFrameElement>();

const state = reactive<LocalState>({
  isEditing: false,
  isUpdating: false,
  description: plan.value.description,
});

const inputRef = ref<InstanceType<typeof NInput>>();

const allowEdit = computed(() => {
  if (isCreating.value) {
    return true;
  }
  // Allowed if current user is the creator.
  if (extractUserId(plan.value.creator) === currentUser.value.email) {
    return true;
  }
  // Allowed if current user has related permission.
  if (hasProjectPermissionV2(project.value, "bb.plans.update")) {
    return true;
  }
  return false;
});

const onDescriptionChange = (value: string) => {
  if (!isCreating.value) {
    return;
  }

  plan.value.description = value;
};

const beginEdit = () => {
  state.description = plan.value.description;
  state.isEditing = true;
  nextTick(() => {
    inputRef.value?.focus();
  });
};

const saveEdit = async () => {
  try {
    state.isUpdating = true;
    const planPatch = Plan.fromPartial({
      ...plan.value,
      description: state.description,
    });
    const updated = await planServiceClient.updatePlan({
      plan: planPatch,
      updateMask: ["description"],
    });
    Object.assign(plan.value, updated);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
    state.isEditing = false;
  } finally {
    state.isUpdating = false;
  }
};

const cancelEdit = () => {
  state.description = plan.value.description;
  state.isEditing = false;
};

const { renderedContent } = useRenderMarkdown(
  computed(() => plan.value.description),
  contentPreviewArea,
  project
);

// Reset the edit state after creating the plan.
watch(isCreating, (curr, prev) => {
  if (!curr && prev) {
    state.isEditing = false;
  }
});

watch(
  () => plan.value,
  (plan) => {
    if (state.isEditing) return;
    state.description = plan.description;
  },
  { immediate: true }
);
</script>
