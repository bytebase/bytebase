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
  DefaultApprovalPolicy,
  DefaultSchedulePolicy,
  PipelineApprovalPolicyPayload,
  BackupPlanPolicyPayload,
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

interface LocalState {
  environment: Environment;
  showArchiveModal: boolean;
  approvalPolicy?: Policy;
  backupPolicy?: Policy;
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
      const environmentId = (state.environment as Environment).id;

      policyStore
        .fetchPolicyByEnvironmentAndType({
          environmentId,
          type: "bb.policy.pipeline-approval",
        })
        .then((policy) => {
          state.approvalPolicy = policy;
        });

      policyStore
        .fetchPolicyByEnvironmentAndType({
          environmentId,
          type: "bb.policy.backup-plan",
        })
        .then((policy) => {
          state.backupPolicy = policy;
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

    const updatePolicy = (
      environmentId: EnvironmentId,
      type: PolicyType,
      policy: Policy
    ) => {
      if (
        type === "bb.policy.pipeline-approval" &&
        (policy.payload as PipelineApprovalPolicyPayload).value !==
          DefaultApprovalPolicy &&
        !hasFeature("bb.feature.approval-policy")
      ) {
        state.missingRequiredFeature = "bb.feature.approval-policy";
        return;
      }
      if (
        type === "bb.policy.backup-plan" &&
        (policy.payload as BackupPlanPolicyPayload).schedule !==
          DefaultSchedulePolicy &&
        !hasFeature("bb.feature.backup-policy")
      ) {
        state.missingRequiredFeature = "bb.feature.backup-policy";
        return;
      }
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
          if (type === "bb.policy.pipeline-approval") {
            state.approvalPolicy = policy;
          } else if (type === "bb.policy.backup-plan") {
            state.backupPolicy = policy;
          } else if (type === "bb.policy.environment-tier") {
            state.environmentTierPolicy = policy;
            // Write the value to state.environment entity. So that we don't
            // need to re-fetch it front the server.
            state.environment.tier = (
              policy.payload as EnvironmentTierPolicyPayload
            ).environmentTier;
            // Also upsert the environment's access-control policy
            const disallowed = state.environment.tier === "PROTECTED";
            await policyStore.upsertPolicyByEnvironmentAndType({
              environmentId: state.environment.id,
              type: "bb.policy.access-control",
              policyUpsert: {
                inheritFromParent: false,
                payload: {
                  disallowRuleList: [{ fullDatabase: disallowed }],
                },
              },
            });
          }
          success();

          if (type === "bb.policy.backup-plan") {
            const payload = state.backupPolicy!
              .payload as BackupPlanPolicyPayload;
            if (payload.schedule === "UNSET") {
              // Changing backup policy from "DAILY"|"WEEKLY" to "UNSET"
              state.showDisableAutoBackupModal = true;
            }
          }
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
      disableAutoBackupContent,
      disableEnvironmentAutoBackup,
    };
  },
});
</script>
