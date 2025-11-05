<template>
  <div
    v-if="create"
    class="flex justify-end items-center gap-x-3"
    v-bind="$attrs"
  >
    <NButton @click.prevent="events.emit('cancel')">
      {{ t("common.cancel") }}
    </NButton>
    <NButton
      type="primary"
      :disabled="!allowCreate"
      @click.prevent="createEnvironment"
    >
      {{ t("common.create") }}
    </NButton>
  </div>

  <div
    v-if="!create && allowEdit && valueChanged()"
    class="flex items-center justify-between gap-x-3"
    v-bind="$attrs"
  >
    <NButton @click.prevent="revertEnvironment">
      {{ t("common.cancel") }}
    </NButton>
    <NButton type="primary" @click.prevent="updateEnvironment">
      {{ t("common.confirm-and-update") }}
    </NButton>
  </div>
</template>

<script setup lang="ts">
import { cloneDeep, isEqual } from "lodash-es";
import { NButton } from "naive-ui";
import { useI18n } from "vue-i18n";
import { pushNotification } from "@/store";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { useEnvironmentFormContext } from "./context";

const { t } = useI18n();
const {
  state,
  environment,
  rolloutPolicy,
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
  events.emit("revert-access-control");
  events.emit("revert-sql-review");
};

const createEnvironment = () => {
  events.emit("create", {
    environment: {
      id: resourceIdField.value?.resourceId,
      title: state.value.environment.title,
      color: state.value.environment.color,
      tags: state.value.environment.tags,
      order: state.value.environment.order,
    },
    rolloutPolicy: state.value.rolloutPolicy,
  });
};

const updateEnvironment = () => {
  if (!isEqual(rolloutPolicy.value, state.value.rolloutPolicy)) {
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
    !isEqual(state.value.environment.tags, env.tags) ||
    state.value.environment.color !== env.color
  ) {
    const environmentPatch = {
      ...env,
      title: state.value.environment.title,
      tags: state.value.environment.tags,
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
