<template>
  <div class="flex-1">
    <MarkdownEditor
      :content="state.description"
      mode="editor"
      :project="project"
      :placeholder="$t('plan.description.placeholder')"
      :issue-list="[]"
      @change="onUpdateValue"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import MarkdownEditor from "@/components/MarkdownEditor";
import { planServiceClientConnect } from "@/grpcweb";
import {
  pushNotification,
  useCurrentUserV1,
  extractUserId,
  useCurrentProjectV1,
} from "@/store";
import { UpdatePlanRequestSchema } from "@/types/proto-es/v1/plan_service_pb";
import { PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContext } from "../../logic";

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const { isCreating, plan, readonly } = usePlanContext();

const state = reactive({
  isUpdating: false,
  description: plan.value.description,
});

const allowEdit = computed(() => {
  if (readonly.value) {
    return false;
  }
  if (isCreating.value) {
    return true;
  }
  if (extractUserId(plan.value.creator) === currentUser.value.email) {
    return true;
  }
  if (hasProjectPermissionV2(project.value, "bb.plans.update")) {
    return true;
  }
  return false;
});

const onUpdateValue = (value: string) => {
  state.description = value;
  if (isCreating.value) {
    plan.value.description = value;
  }
};

let debounceTimer: NodeJS.Timeout | null = null;

watch(
  () => state.description,
  (newValue) => {
    if (isCreating.value || !allowEdit.value) {
      return;
    }

    if (newValue === plan.value.description) {
      return;
    }

    if (debounceTimer) {
      clearTimeout(debounceTimer);
    }

    debounceTimer = setTimeout(async () => {
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
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      } finally {
        state.isUpdating = false;
      }
    }, 1000);
  }
);

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
