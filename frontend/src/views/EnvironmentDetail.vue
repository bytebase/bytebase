<template>
  <EnvironmentForm
    v-if="state.rolloutPolicy"
    :environment="state.environment"
    :rollout-policy="state.rolloutPolicy"
    @update="doUpdate"
    @delete="doDelete"
    @update-policy="updatePolicy"
  >
    <EnvironmentFormBody :features="features" class="w-full px-4" />
    <EnvironmentFormButtons
      class="sticky -bottom-4 bg-white py-4 px-2 border-t border-block-border"
      :class="buttonsClass"
    />
  </EnvironmentForm>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { computed, reactive, watch, watchEffect } from "vue";
import {
  EnvironmentForm,
  Form as EnvironmentFormBody,
  Buttons as EnvironmentFormButtons,
} from "@/components/EnvironmentForm";
import { useEnvironmentV1Store } from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import {
  usePolicyV1Store,
  getEmptyRolloutPolicy,
} from "@/store/modules/v1/policy";
import { formatEnvironmentName } from "@/types";
import type { Policy } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import type { Environment } from "@/types/v1/environment";
import { type VueClass } from "@/utils";

interface LocalState {
  environment: Environment;
  showArchiveModal: boolean;
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

const state = reactive<LocalState>({
  environment: environmentV1Store.getEnvironmentByName(
    `${environmentNamePrefix}${props.environmentName}`
  ),
  showArchiveModal: false,
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

const preparePolicy = () => {
  policyV1Store
    .fetchPolicies({
      parent: stateEnvironmentName.value,
      resourceType: PolicyResourceType.ENVIRONMENT,
    })
    .then((policies) => {
      const rolloutPolicy = policies.find(
        (policy) => policy.type === PolicyType.ROLLOUT_POLICY
      );
      state.rolloutPolicy =
        rolloutPolicy ||
        getEmptyRolloutPolicy(
          stateEnvironmentName.value,
          PolicyResourceType.ENVIRONMENT
        );
    });
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
</script>
