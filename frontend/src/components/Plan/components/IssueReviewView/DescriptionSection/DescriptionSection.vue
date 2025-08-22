<template>
  <div class="space-y-2">
    <div class="flex items-center justify-between gap-2">
      <h3 class="text-base font-medium">
        {{ $t("common.description") }}
      </h3>
    </div>
    <NInput
      v-model:value="state.description"
      type="textarea"
      :placeholder="$t('issue.add-some-description')"
      :disabled="!allowEdit"
      :autosize="false"
      :resizable="false"
      @blur="onPlanDescriptionUpdate"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { NInput } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import { planServiceClientConnect } from "@/grpcweb";
import {
  extractUserId,
  pushNotification,
  useCurrentProjectV1,
  useCurrentUserV1,
} from "@/store";
import {
  PlanSchema,
  UpdatePlanRequestSchema,
} from "@/types/proto-es/v1/plan_service_pb";
import { hasProjectPermissionV2 } from "@/utils";
import { usePlanContext } from "../../..";

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const { plan, readonly, isCreating } = usePlanContext();
const { refreshResources } = useResourcePoller();

const state = reactive({
  description: plan.value.description,
});

// Watch for changes in plan to update the description
watch(
  () => [plan.value],
  () => {
    state.description = plan.value.description;
  },
  { immediate: true }
);

const allowEdit = computed(() => {
  if (readonly.value) {
    return false;
  }
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

const onPlanDescriptionUpdate = async () => {
  // Only update if description actually changed
  if (state.description === plan.value.description) {
    return;
  }

  const request = create(UpdatePlanRequestSchema, {
    plan: create(PlanSchema, {
      name: plan.value.name,
      description: state.description,
    }),
    updateMask: { paths: ["description"] },
  });
  await planServiceClientConnect.updatePlan(request);
  refreshResources(["plan"], true /** force */);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
};
</script>
