<template>
  <div v-if="state.environment.state == State.DELETED" class="mb-2 -mt-4">
    <ArchiveBanner />
  </div>

  <EnvironmentForm
    v-if="state.rolloutPolicy && state.environmentTier"
    :environment="state.environment"
    :rollout-policy="state.rolloutPolicy"
    :environment-tier="state.environmentTier"
    @update="doUpdate"
    @archive="doArchive"
    @restore="doRestore"
    @update-policy="updatePolicy"
  >
    <EnvironmentFormBody
      :simple="simple"
      :hide-archive-restore="hideArchiveRestore"
      class="w-full px-4 pb-2"
      :class="bodyClass"
    />
    <EnvironmentFormButtons
      class="sticky bottom-0 bg-white py-2 px-4 border-t border-block-border"
      :class="buttonsClass"
    />
  </EnvironmentForm>

  <FeatureModal
    :open="state.missingRequiredFeature != undefined"
    :feature="state.missingRequiredFeature"
    @cancel="state.missingRequiredFeature = undefined"
  />
</template>

<script lang="ts" setup>
import { cloneDeep } from "lodash-es";
import { reactive, watch, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import ArchiveBanner from "@/components/ArchiveBanner.vue";
import {
  EnvironmentForm,
  Form as EnvironmentFormBody,
  Buttons as EnvironmentFormButtons,
} from "@/components/EnvironmentForm";
import { FeatureModal } from "@/components/FeatureGuard";
import { ENVIRONMENT_V1_ROUTE_DETAIL } from "@/router/dashboard/environmentV1";
import { ENVIRONMENT_V1_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { hasFeature, pushNotification } from "@/store";
import { environmentNamePrefix } from "@/store/modules/v1/common";
import {
  useEnvironmentV1Store,
  defaultEnvironmentTier,
} from "@/store/modules/v1/environment";
import {
  usePolicyV1Store,
  getEmptyRolloutPolicy,
} from "@/store/modules/v1/policy";
import { VirtualRoleType, unknownEnvironment } from "@/types";
import { State } from "@/types/proto/v1/common";
import type {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import type { Policy } from "@/types/proto/v1/org_policy_service";
import {
  PolicyType,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { extractEnvironmentResourceName, type VueClass } from "@/utils";

interface LocalState {
  environment: Environment;
  showArchiveModal: boolean;
  rolloutPolicy?: Policy;
  environmentTier?: EnvironmentTier;
  missingRequiredFeature?:
    | "bb.feature.approval-policy"
    | "bb.feature.custom-approval"
    | "bb.feature.environment-tier-policy";
}

const props = defineProps<{
  environmentName: string;
  simple?: boolean;
  hideArchiveRestore?: boolean;
  bodyClass?: VueClass;
  buttonsClass?: VueClass;
}>();

const emit = defineEmits(["archive"]);

const { t } = useI18n();
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
      parent: state.environment.name,
      resourceType: PolicyResourceType.ENVIRONMENT,
    })
    .then((policies) => {
      const rolloutPolicy = policies.find(
        (policy) => policy.type === PolicyType.ROLLOUT_POLICY
      );
      state.rolloutPolicy =
        rolloutPolicy ||
        getEmptyRolloutPolicy(
          state.environment.name,
          PolicyResourceType.ENVIRONMENT
        );
    });

  state.environmentTier = state.environment.tier;
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
  if (environmentPatch.tier !== pendingUpdate.tier) {
    if (
      environmentPatch.tier !== defaultEnvironmentTier &&
      !hasFeature("bb.feature.environment-tier-policy")
    ) {
      state.missingRequiredFeature = "bb.feature.environment-tier-policy";
      return;
    }
    pendingUpdate.tier = environmentPatch.tier;
  }

  environmentV1Store
    .updateEnvironment(pendingUpdate)
    .then((environment) => {
      assignEnvironment(environment);
    })
    .then(() => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("environment.successfully-updated-environment", {
          name: state.environment.title,
        }),
      });
    });
};

const doArchive = (environment: Environment) => {
  environmentV1Store.deleteEnvironment(environment.name).then(() => {
    emit("archive", environment);
    environment.state = State.DELETED;
    assignEnvironment(environment);
    router.replace({
      name: ENVIRONMENT_V1_ROUTE_DETAIL,
      params: {
        environmentName: extractEnvironmentResourceName(environment.name),
      },
    });
  });
};

const doRestore = (environment: Environment) => {
  environmentV1Store
    .undeleteEnvironment(environment.name)
    .then((environment) => {
      assignEnvironment(environment);
      const id = extractEnvironmentResourceName(environment.name);
      router.replace({
        name: ENVIRONMENT_V1_ROUTE_DASHBOARD,
        hash: `#${id}`,
      });
    });
};

const success = () => {
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("environment.successfully-updated-environment", {
      name: state.environment.title,
    }),
  });
};

const updatePolicy = async (params: {
  environment: Environment;
  policyType: PolicyType;
  policy: Policy;
}) => {
  const { environment, policyType, policy } = params;
  if (policyType === PolicyType.ROLLOUT_POLICY) {
    const rp = policy.rolloutPolicy;
    if (rp?.automatic === false) {
      if (rp.issueRoles.includes(VirtualRoleType.LAST_APPROVER)) {
        if (!hasFeature("bb.feature.custom-approval")) {
          state.missingRequiredFeature = "bb.feature.custom-approval";
          return;
        }
      }
      if (!hasFeature("bb.feature.approval-policy")) {
        state.missingRequiredFeature = "bb.feature.approval-policy";
        return;
      }
    }
  }

  const updatedPolicy = await policyV1Store.upsertPolicy({
    parentPath: environment.name,
    updateMask: ["payload"],
    policy,
  });
  switch (policyType) {
    case PolicyType.ROLLOUT_POLICY:
      state.rolloutPolicy = updatedPolicy;
      break;
  }

  success();
};
</script>
