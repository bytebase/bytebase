<template>
  <div class="py-2">
    <ArchiveBanner v-if="state.environment.rowStatus == 'ARCHIVED'" />
  </div>
  <EnvironmentForm
    v-if="
      state.approvalPolicy && state.backupPolicy && state.environmentTierPolicy
    "
    :environment="state.environment"
    :approval-policy="state.approvalPolicy"
    :backup-policy="state.backupPolicy"
    :environment-tier-policy="state.environmentTierPolicy"
    @update="doUpdate"
    @archive="doArchive"
    @restore="doRestore"
    @update-policy="updatePolicy"
    @update-policy-v1="updatePolicyV1"
  />
  <FeatureModal
    v-if="state.missingRequiredFeature != undefined"
    :feature="state.missingRequiredFeature"
    @cancel="state.missingRequiredFeature = undefined"
  />
  <BBModal
    v-if="state.showDisableAutoBackupModal"
    :title="$t('environment.disable-auto-backup.self')"
    @close="state.showDisableAutoBackupModal = false"
  >
    <div class="space-y-2 textinfolabel pr-16">
      <p v-for="(line, i) in disableAutoBackupContent.split('\n')" :key="i">
        {{ line }}
      </p>
    </div>

    <div class="flex items-center justify-end pt-4 mt-4 border-t">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click="state.showDisableAutoBackupModal = false"
      >
        {{ $t("common.no") }}
      </button>
      <button
        type="submit"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        @click="disableEnvironmentAutoBackup"
      >
        {{ $t("common.yes") }}
      </button>
    </div>
  </BBModal>
</template>

<script lang="ts">
import { computed, defineComponent, reactive, watchEffect } from "vue";
import ArchiveBanner from "../components/ArchiveBanner.vue";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import {
  Environment,
  EnvironmentId,
  EnvironmentPatch,
  Policy,
  PolicyType,
  EnvironmentTierPolicyPayload,
  DefaultEnvironmentTier,
} from "../types";
import { idFromSlug } from "../utils";
import {
  hasFeature,
  pushNotification,
  useBackupStore,
  useEnvironmentStore,
  usePolicyStore,
} from "@/store";
import { useI18n } from "vue-i18n";
import BBModal from "@/bbkit/BBModal.vue";
import {
  usePolicyV1Store,
  defaultBackupSchedule,
  defaultApprovalStrategy,
  getDefaultBackupPlanPolicy,
  getDefaultDeploymentApprovalPolicy,
} from "@/store/modules/v1/policy";
import { getEnvironmentPathByLegacyEnvironment } from "@/store/modules/v1/common";
import {
  Policy as PolicyV1,
  PolicyType as PolicyTypeV1,
  PolicyResourceType,
  BackupPlanSchedule,
} from "@/types/proto/v1/org_policy_service";

interface LocalState {
  environment: Environment;
  showArchiveModal: boolean;
  approvalPolicy?: PolicyV1;
  backupPolicy?: PolicyV1;
  environmentTierPolicy?: Policy;
  missingRequiredFeature?:
    | "bb.feature.approval-policy"
    | "bb.feature.backup-policy"
    | "bb.feature.environment-tier-policy";
  showDisableAutoBackupModal: boolean;
}

export default defineComponent({
  name: "EnvironmentDetail",
  components: {
    ArchiveBanner,
    EnvironmentForm,
    BBModal,
  },
  props: {
    environmentSlug: {
      required: true,
      type: String,
    },
  },
  emits: ["archive"],
  setup(props, { emit }) {
    const environmentStore = useEnvironmentStore();
    const policyStore = usePolicyStore();
    const policyV1Store = usePolicyV1Store();
    const backupStore = useBackupStore();
    const { t } = useI18n();

    const state = reactive<LocalState>({
      environment: environmentStore.getEnvironmentById(
        idFromSlug(props.environmentSlug)
      ),
      showArchiveModal: false,
      showDisableAutoBackupModal: false,
    });

    const preparePolicy = () => {
      const environmentId = state.environment.id;

      policyV1Store
        .getOrFetchPolicyByParentAndType({
          parentPath: getEnvironmentPathByLegacyEnvironment(state.environment),
          policyType: PolicyTypeV1.DEPLOYMENT_APPROVAL,
        })
        .then((policy) => {
          state.approvalPolicy =
            policy ||
            getDefaultDeploymentApprovalPolicy(
              getEnvironmentPathByLegacyEnvironment(state.environment),
              PolicyResourceType.ENVIRONMENT
            );
        });

      policyV1Store
        .getOrFetchPolicyByParentAndType({
          parentPath: getEnvironmentPathByLegacyEnvironment(state.environment),
          policyType: PolicyTypeV1.BACKUP_PLAN,
        })
        .then((policy) => {
          state.backupPolicy =
            policy ||
            getDefaultBackupPlanPolicy(
              getEnvironmentPathByLegacyEnvironment(state.environment),
              PolicyResourceType.ENVIRONMENT
            );
        });

      policyStore
        .fetchPolicyByEnvironmentAndType({
          environmentId,
          type: "bb.policy.environment-tier",
        })
        .then((policy) => {
          state.environmentTierPolicy = policy;
        });
    };

    watchEffect(preparePolicy);

    const assignEnvironment = (environment: Environment) => {
      state.environment = environment;
    };

    const doUpdate = (environmentPatch: EnvironmentPatch) => {
      environmentStore
        .patchEnvironment({
          environmentId: idFromSlug(props.environmentSlug),
          environmentPatch,
        })
        .then((environment) => {
          assignEnvironment(environment);

          pushNotification({
            module: "bytebase",
            style: "SUCCESS",
            title: t("environment.successfully-updated-environment", {
              name: environment.name,
            }),
          });
        });
    };

    const doArchive = (environment: Environment) => {
      environmentStore
        .patchEnvironment({
          environmentId: environment.id,
          environmentPatch: {
            rowStatus: "ARCHIVED",
          },
        })
        .then((environment) => {
          emit("archive", environment);
          assignEnvironment(environment);
        });
    };

    const doRestore = (environment: Environment) => {
      environmentStore
        .patchEnvironment({
          environmentId: environment.id,
          environmentPatch: {
            rowStatus: "NORMAL",
          },
        })
        .then((environment) => {
          assignEnvironment(environment);
        });
    };

    const success = () => {
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("environment.successfully-updated-environment", {
          name: state.environment.name,
        }),
      });
    };

    const updatePolicyV1 = async (
      environment: Environment,
      policyType: PolicyTypeV1,
      policy: PolicyV1
    ) => {
      switch (policyType) {
        case PolicyTypeV1.DEPLOYMENT_APPROVAL:
          if (
            policy.deploymentApprovalPolicy?.defaultStrategy !=
              defaultApprovalStrategy &&
            !hasFeature("bb.feature.approval-policy")
          ) {
            state.missingRequiredFeature = "bb.feature.approval-policy";
            return;
          }
          break;
        case PolicyTypeV1.BACKUP_PLAN:
          if (
            policy.backupPlanPolicy?.schedule != defaultBackupSchedule &&
            !hasFeature("bb.feature.backup-policy")
          ) {
            state.missingRequiredFeature = "bb.feature.backup-policy";
            return;
          }
          break;
      }

      const updatedPolicy = await policyV1Store.upsertPolicy({
        parentPath: getEnvironmentPathByLegacyEnvironment(environment),
        updateMask: ["payload"],
        policy,
      });
      switch (policyType) {
        case PolicyTypeV1.DEPLOYMENT_APPROVAL:
          state.approvalPolicy = updatedPolicy;
          break;
        case PolicyTypeV1.BACKUP_PLAN:
          state.backupPolicy = updatedPolicy;
          break;
      }

      success();
      if (policyType === PolicyTypeV1.BACKUP_PLAN) {
        if (
          state.backupPolicy?.backupPlanPolicy?.schedule ==
          BackupPlanSchedule.UNSET
        ) {
          // Changing backup policy from "DAILY"|"WEEKLY" to "UNSET"
          state.showDisableAutoBackupModal = true;
        }
      }
    };

    const updatePolicy = (
      environmentId: EnvironmentId,
      type: PolicyType,
      policy: Policy
    ) => {
      if (
        type === "bb.policy.environment-tier" &&
        (policy.payload as EnvironmentTierPolicyPayload).environmentTier !==
          DefaultEnvironmentTier &&
        !hasFeature("bb.feature.environment-tier-policy")
      ) {
        state.missingRequiredFeature = "bb.feature.environment-tier-policy";
        return;
      }
      policyStore
        .upsertPolicyByEnvironmentAndType({
          environmentId,
          type: type,
          policyUpsert: {
            payload: policy.payload,
          },
        })
        .then(async (policy: Policy) => {
          if (type === "bb.policy.environment-tier") {
            state.environmentTierPolicy = policy;
            // Write the value to state.environment entity. So that we don't
            // need to re-fetch it front the server.
            state.environment.tier = (
              policy.payload as EnvironmentTierPolicyPayload
            ).environmentTier;
            // Also upsert the environment's access-control policy
            const disallowed = state.environment.tier === "PROTECTED";
            await policyV1Store.upsertPolicy({
              parentPath: getEnvironmentPathByLegacyEnvironment(
                state.environment
              ),
              updateMask: ["payload", "inherit_from_parent"],
              policy: {
                type: PolicyTypeV1.ACCESS_CONTROL,
                inheritFromParent: false,
                accessControlPolicy: {
                  disallowRules: [{ fullDatabase: disallowed }],
                },
              },
            });
          }
          success();
        });
    };

    const disableAutoBackupContent = computed(() => {
      return t("environment.disable-auto-backup.content");
    });

    const disableEnvironmentAutoBackup = async () => {
      await backupStore.upsertBackupSettingByEnvironmentId(
        state.environment.id,
        {
          enabled: false,
          hour: 0,
          dayOfWeek: 0,
          retentionPeriodTs: 0,
          hookUrl: "",
        }
      );
      success();
      state.showDisableAutoBackupModal = false;
    };

    return {
      state,
      doUpdate,
      doArchive,
      doRestore,
      updatePolicy,
      updatePolicyV1,
      disableAutoBackupContent,
      disableEnvironmentAutoBackup,
    };
  },
});
</script>
