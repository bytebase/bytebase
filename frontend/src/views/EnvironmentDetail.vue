<template>
  <EnvironmentForm
    v-if="state.rolloutPolicy"
    ref="environmentFormRef"
    :environment="state.environment"
    :rollout-policy="state.rolloutPolicy"
    @update="doUpdate"
    @delete="doDelete"
    @update-policy="updatePolicy"
  >
    <div class="flex flex-col h-full w-full">
      <EnvironmentFormBody :features="features" class="w-full flex-1 px-4" />
      <EnvironmentFormButtons
        class="sticky -bottom-4 bg-white py-4 px-2 border-t border-block-border"
        :class="buttonsClass"
      />
    </div>
  </EnvironmentForm>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { computed, reactive, ref, watch, watchEffect } from "vue";
import {
  EnvironmentForm,
  Form as EnvironmentFormBody,
  Buttons as EnvironmentFormButtons,
} from "@/components/EnvironmentForm";
import { useEnvironmentV1Store } from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { usePolicyV1Store } from "@/store/modules/v1/policy";
import { formatEnvironmentName } from "@/types";
import type { Policy } from "@/types/proto-es/v1/org_policy_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import type { Environment } from "@/types/v1/environment";
import { type VueClass } from "@/utils";

interface LocalState {
  environment: Environment;
  rolloutPolicy?: Policy;
}

const props = defineProps<{
  features?: InstanceType<typeof EnvironmentFormBody>["features"];
  environmentName: string;
  buttonsClass?: VueClass;
}>();

const emit = defineEmits<{
  (event: "delete", environment: Environment): void;
}>();

const environmentV1Store = useEnvironmentV1Store();
const policyV1Store = usePolicyV1Store();
const environmentFormRef = ref<InstanceType<typeof EnvironmentForm>>();

const state = reactive<LocalState>({
  environment: environmentV1Store.getEnvironmentByName(
    `${environmentNamePrefix}${props.environmentName}`
  ),
});

const stateEnvironmentName = computed(() => {
  return formatEnvironmentName(state.environment.id);
});

const prepareEnvironment = async () => {
  await environmentV1Store.getOrFetchEnvironmentByName(
    `${environmentNamePrefix}${props.environmentName}`
  );
};

watch(() => props.environmentName, prepareEnvironment, {
  immediate: true,
});

// Fetch rollout policy directly using getOrFetchPolicyByParentAndType
// The backend now returns default policy if none exists, so no manual fallback needed
const preparePolicy = async () => {
  const policy = await policyV1Store.getOrFetchPolicyByParentAndType({
    parentPath: stateEnvironmentName.value,
    policyType: PolicyType.ROLLOUT_POLICY,
  });
  state.rolloutPolicy = policy;
};

watchEffect(preparePolicy);

const assignEnvironment = (environment: Environment) => {
  state.environment = environment;
};

const doUpdate = (environmentPatch: Environment) => {
  const pendingUpdate = cloneDeep(state.environment);
  if (environmentPatch.title !== pendingUpdate.title) {
    pendingUpdate.title = environmentPatch.title;
  }
  if (!isEqual(environmentPatch.tags, pendingUpdate.tags)) {
    pendingUpdate.tags = environmentPatch.tags;
  }
  if (environmentPatch.color !== pendingUpdate.color) {
    pendingUpdate.color = environmentPatch.color;
  }

  environmentV1Store.updateEnvironment(pendingUpdate).then((environment) => {
    assignEnvironment(environment);
  });
};

const doDelete = (environment: Environment) => {
  emit("delete", environment);
};

const updatePolicy = async (params: {
  environment: Environment;
  policyType: PolicyType;
  policy: Policy;
}) => {
  const { environment, policyType, policy } = params;

  const updatedPolicy = await policyV1Store.upsertPolicy({
    parentPath: formatEnvironmentName(environment.id),
    policy,
  });
  switch (policyType) {
    case PolicyType.ROLLOUT_POLICY:
      state.rolloutPolicy = updatedPolicy;
      break;
  }
};

defineExpose({
  isEditing: computed(() => environmentFormRef.value?.isEditing ?? false),
});
</script>
