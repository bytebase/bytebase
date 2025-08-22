<template>
  <div class="flex-1">
    <NInput
      v-model:value="state.description"
      :placeholder="$t('issue.add-some-description')"
      :disabled="!allowEdit || state.isUpdating"
      :loading="state.isUpdating"
      :style="style"
      size="tiny"
      type="textarea"
      :autosize="{ minRows: 1, maxRows: 3 }"
      @focus="state.isFocused = true"
      @blur="onBlur"
      @update:value="onUpdateValue"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NInput } from "naive-ui";
import type { CSSProperties } from "vue";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
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
  isFocused: false,
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

const style = computed(() => {
  const style: CSSProperties = {
    "--n-color-disabled": "transparent",
  };
  const border = state.isFocused
    ? "1px solid rgb(var(--color-control-border))"
    : "none";
  style["--n-border"] = border;
  style["--n-border-disabled"] = border;

  return style;
});

const onBlur = async () => {
  state.isFocused = false;
  if (isCreating.value) {
    return;
  }

  if (state.description === plan.value.description) {
    return;
  }

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
};

const onUpdateValue = (value: string) => {
  if (!isCreating.value) {
    return;
  }
  plan.value.description = value;
};
</script>
