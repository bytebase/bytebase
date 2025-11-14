<template>
  <slot />

  <FeatureModal
    :open="missingFeature != undefined"
    :feature="missingFeature"
    @cancel="missingFeature = undefined"
  />
</template>

<script lang="ts" setup>
import { computed, toRef } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { hasFeature } from "@/store";
import type { Policy } from "@/types/proto-es/v1/org_policy_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import type { Environment } from "@/types/v1/environment";
import { FeatureModal } from "../FeatureGuard";
import { provideEnvironmentFormContext } from "./context";

const props = withDefaults(
  defineProps<{
    create?: boolean;
    environment: Environment;
    rolloutPolicy: Policy;
  }>(),
  {
    create: false,
  }
);

const emit = defineEmits<{
  (
    event: "create",
    params: {
      environment: Partial<Environment>;
      rolloutPolicy: Policy;
    }
  ): void;
  (event: "update", environment: Environment): void;
  (
    event: "update-policy",
    params: {
      environment: Environment;
      policyType: PolicyType;
      policy: Policy;
    }
  ): void;
  (event: "delete", environment: Environment): void;
  (event: "cancel"): void;
}>();

const context = provideEnvironmentFormContext({
  create: toRef(props, "create"),
  environment: toRef(props, "environment"),
  rolloutPolicy: toRef(props, "rolloutPolicy"),
});
const { valueChanged, events, missingFeature } = context;

const isEditing = computed(() => {
  return !props.create && valueChanged();
});

useEmitteryEventListener(events, "create", (params) => {
  const { environment } = params;
  if (environment.tags?.protected === "protected") {
    if (!hasFeature(PlanFeature.FEATURE_ENVIRONMENT_TIERS)) {
      missingFeature.value = PlanFeature.FEATURE_ENVIRONMENT_TIERS;
      return;
    }
  }

  emit("create", params);
});
useEmitteryEventListener(events, "update", (environment) => {
  if (
    environment.tags.protected === "protected" &&
    !hasFeature(PlanFeature.FEATURE_ENVIRONMENT_TIERS)
  ) {
    missingFeature.value = PlanFeature.FEATURE_ENVIRONMENT_TIERS;
    return;
  }

  emit("update", environment);
});
useEmitteryEventListener(events, "update-policy", (params) => {
  emit("update-policy", params);
});
useEmitteryEventListener(events, "delete", (environment) => {
  emit("delete", environment);
});
useEmitteryEventListener(events, "cancel", () => {
  emit("cancel");
});

defineExpose({
  isEditing,
});
</script>
