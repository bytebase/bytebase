<template>
  <div class="py-2">
    <ArchiveBanner v-if="state.environment.state == State.DELETED" />
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
  />
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
import EnvironmentForm from "@/components/EnvironmentForm.vue";
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
import {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import {
  Policy as PolicyV1,
  PolicyType as PolicyTypeV1,
  PolicyResourceType,
} from "@/types/proto/v1/org_policy_service";
import { extractEnvironmentResourceName } from "@/utils";

interface LocalState {
  environment: Environment;
  showArchiveModal: boolean;
  rolloutPolicy?: PolicyV1;
  environmentTier?: EnvironmentTier;
  missingRequiredFeature?:
    | "bb.feature.approval-policy"
    | "bb.feature.custom-approval"
    | "bb.feature.environment-tier-policy";
}

const props = defineProps({
  environmentId: {
    required: true,
    type: String,
  },
});

const emit = defineEmits(["archive"]);

const router = useRouter();
const { t } = useI18n();
const environmentV1Store = useEnvironmentV1Store();
const policyV1Store = usePolicyV1Store();

const state = reactive<LocalState>({
  environment:
    environmentV1Store.getEnvironmentByName(
      `${environmentNamePrefix}${props.environmentId}`
    ) || unknownEnvironment(),
  showArchiveModal: false,
});

const prepareEnvironment = async () => {
  await environmentV1Store.getOrFetchEnvironmentByName(
    `${environmentNamePrefix}${props.environmentId}`
  );
};

watch(() => props.environmentId, prepareEnvironment, {
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
        (policy) => policy.type === PolicyTypeV1.ROLLOUT_POLICY
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
        environmentId: extractEnvironmentResourceName(environment.name),
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

const updatePolicy = async (
  environment: Environment,
  policyType: PolicyTypeV1,
  policy: PolicyV1
) => {
  if (policyType === PolicyTypeV1.ROLLOUT_POLICY) {
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
    case PolicyTypeV1.ROLLOUT_POLICY:
      state.rolloutPolicy = updatedPolicy;
      break;
  }

  success();
};
</script>
