<template>
  <EnvironmentForm
    v-if="state.rolloutPolicy"
    :environment="state.environment"
    :rollout-policy="state.rolloutPolicy"
    @update="doUpdate"
    @archive="doDelete"
    @update-policy="updatePolicy"
  >
    <EnvironmentFormBody
      :features="features"
      :hide-archive-restore="hideArchiveRestore"
      class="w-full px-4 pb-4"
      :class="bodyClass"
    />
    <EnvironmentFormButtons
      class="sticky bottom-0 bg-white py-4 px-2 border-t border-block-border"
      :class="buttonsClass"
    />
  </EnvironmentForm>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { computed, reactive, watch, watchEffect } from "vue";
import { useRouter } from "vue-router";
import {
  EnvironmentForm,
  Form as EnvironmentFormBody,
  Buttons as EnvironmentFormButtons,
} from "@/components/EnvironmentForm";
import { ENVIRONMENT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import { useEnvironmentV1Store } from "@/store/modules/v1/environment";
import {
  usePolicyV1Store,
  getEmptyRolloutPolicy,
} from "@/store/modules/v1/policy";
import { formatEnvironmentName, unknownEnvironment } from "@/types";
import type { Policy } from "@/types/proto/v1/org_policy_service";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
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
  hideArchiveRestore?: boolean;
  bodyClass?: VueClass;
  buttonsClass?: VueClass;
}>();

const emit = defineEmits(["delete"]);

const router = useRouter();
const environmentV1Store = useEnvironmentV1Store();
const policyV1Store = usePolicyV1Store();

const state = reactive<LocalState>({
  environment:
    environmentV1Store.getEnvironmentByName(
      `${environmentNamePrefix}${props.environmentName}`
    ) || unknownEnvironment(),
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
  environmentV1Store
    .deleteEnvironment(formatEnvironmentName(environment.id))
    .then(() => {
      emit("delete", environment);
      assignEnvironment(environment);
      router.replace({
        name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
      });
    });
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
