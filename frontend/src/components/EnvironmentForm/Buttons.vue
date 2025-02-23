<template>
  <div
    v-if="create"
    class="flex justify-end items-center gap-x-3"
    v-bind="$attrs"
  >
    <NButton @click.prevent="events.emit('cancel')">
      {{ $t("common.cancel") }}
    </NButton>
    <NButton
      type="primary"
      :disabled="!allowCreate"
      @click.prevent="createEnvironment"
    >
      {{ $t("common.create") }}
    </NButton>
  </div>

  <div
    v-if="!create && allowEdit && valueChanged()"
    class="flex items-center justify-between gap-x-3"
    v-bind="$attrs"
  >
    <NButton @click.prevent="revertEnvironment">
      {{ $t("common.cancel") }}
    </NButton>
    <NButton type="primary" @click.prevent="updateEnvironment">
      {{ $t("common.confirm-and-update") }}
    </NButton>
  </div>
</template>

<script setup lang="ts">
import { cloneDeep, isEqual } from "lodash-es";
import { NButton } from "naive-ui";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { useEnvironmentFormContext } from "./context";

const { t } = useI18n();
const {
  state,
  environment,
  rolloutPolicy,
  environmentTier,
  create,
  resourceIdField,
  allowEdit,
  allowCreate,
  valueChanged,
  events,
} = useEnvironmentFormContext();

const revertEnvironment = () => {
  state.value.environment = cloneDeep(environment.value);
  state.value.rolloutPolicy = cloneDeep(rolloutPolicy.value);
  state.value.environmentTier = cloneDeep(environmentTier.value);
  events.emit("revert-access-control");
  events.emit("revert-sql-review");
};

const createEnvironment = () => {
  events.emit("create", {
    environment: {
      name: resourceIdField.value?.resourceId,
      title: state.value.environment.title,
      color: state.value.environment.color,
    },
    rolloutPolicy: state.value.rolloutPolicy,
    environmentTier: state.value.environmentTier,
  });
};

const updateEnvironment = () => {
  if (!isEqual(rolloutPolicy.value, state.value.rolloutPolicy)) {
    // Validate rollout policy.
    if (
      !state.value.rolloutPolicy.rolloutPolicy?.automatic &&
      state.value.rolloutPolicy.rolloutPolicy?.roles.length === 0 &&
      state.value.rolloutPolicy.rolloutPolicy?.issueRoles.length === 0
    ) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("policy.rollout.select-at-least-one-role"),
      });
      return;
    }
    events.emit("update-policy", {
      environment: state.value.environment,
      policyType: PolicyType.ROLLOUT_POLICY,
      policy: state.value.rolloutPolicy,
    });
  }

  events.emit("update-access-control");
  events.emit("update-sql-review");

  const env = cloneDeep(environment.value);
  if (
    state.value.environment.title !== env.title ||
    state.value.environmentTier !== env.tier ||
    state.value.environment.color !== env.color
  ) {
    const environmentPatch = {
      ...env,
      title: state.value.environment.title,
      tier: state.value.environmentTier,
      color: state.value.environment.color,
    };
    events.emit("update", environmentPatch);
  }

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("environment.successfully-updated-environment", {
      name: state.value.environment.title,
    }),
  });
};
</script>
