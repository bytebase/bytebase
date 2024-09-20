<template>
  <slot />

  <FeatureModal
    :open="missingFeature != undefined"
    :feature="missingFeature"
    @cancel="missingFeature = undefined"
  />
</template>

<script lang="ts" setup>
import { useEventListener } from "@vueuse/core";
import { toRef } from "vue";
import { useI18n } from "vue-i18n";
import { onBeforeRouteLeave } from "vue-router";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { hasFeature } from "@/store";
import { VirtualRoleType } from "@/types";
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
const { state, valueChanged, events, missingFeature } = context;

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
  const { rolloutPolicy, environmentTier } = params;
  if (environmentTier === EnvironmentTier.PROTECTED) {
    if (!hasFeature("bb.feature.custom-approval")) {
      missingFeature.value = "bb.feature.environment-tier-policy";
      return;
    }
  }
  const rp = rolloutPolicy.rolloutPolicy;
  if (rp?.automatic === false) {
    if (rp.issueRoles.includes(VirtualRoleType.LAST_APPROVER)) {
      if (!hasFeature("bb.feature.custom-approval")) {
        missingFeature.value = "bb.feature.custom-approval";
        return;
      }
    }
    if (!hasFeature("bb.feature.approval-policy")) {
      missingFeature.value = "bb.feature.approval-policy";
      return;
    }
  }

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
