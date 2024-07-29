<template>
  <slot />

  <FeatureModal
    :open="state.missingRequiredFeature != undefined"
    :feature="state.missingRequiredFeature"
    @cancel="state.missingRequiredFeature = undefined"
  />
</template>

<script lang="ts" setup>
import { useEventListener } from "@vueuse/core";
import { toRef } from "vue";
import { useI18n } from "vue-i18n";
import { onBeforeRouteLeave } from "vue-router";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import type { Environment } from "@/types/proto/v1/environment_service";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import type { Policy } from "@/types/proto/v1/org_policy_service";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { FeatureModal } from "../FeatureGuard";
import { provideEnvironmentFormContext } from "./context";

const props = withDefaults(
  defineProps<{
    create?: boolean;
    environment: Environment;
    rolloutPolicy: Policy;
    environmentTier: EnvironmentTier;
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
      environmentTier: EnvironmentTier;
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
  (event: "archive", environment: Environment): void;
  (event: "restore", environment: Environment): void;
  (event: "cancel"): void;
}>();

const { t } = useI18n();
const context = provideEnvironmentFormContext({
  create: toRef(props, "create"),
  environment: toRef(props, "environment"),
  rolloutPolicy: toRef(props, "rolloutPolicy"),
  environmentTier: toRef(props, "environmentTier"),
});
const { state, valueChanged, events } = context;

useEventListener("beforeunload", (e) => {
  if (props.create || !valueChanged()) {
    return;
  }
  e.returnValue = t("common.leave-without-saving");
  return e.returnValue;
});

onBeforeRouteLeave((to, from, next) => {
  if (!props.create && valueChanged()) {
    if (!window.confirm(t("common.leave-without-saving"))) {
      return;
    }
  }
  next();
});

useEmitteryEventListener(events, "create", (params) => {
  emit("create", params);
});
useEmitteryEventListener(events, "update", (environment) => {
  emit("update", environment);
});
useEmitteryEventListener(events, "update-policy", (params) => {
  emit("update-policy", params);
});
useEmitteryEventListener(events, "archive", (environment) => {
  emit("archive", environment);
});
useEmitteryEventListener(events, "restore", (environment) => {
  emit("restore", environment);
});
useEmitteryEventListener(events, "cancel", () => {
  emit("cancel");
});
</script>
