<template>
  <div class="py-2">
    <ArchiveBanner v-if="state.environment.state == State.DELETED" />
  </div>
  <EnvironmentForm
    v-if="state.approvalPolicy && state.backupPolicy && state.environmentTier"
    :environment="state.environment"
    :approval-policy="state.approvalPolicy"
    :backup-policy="state.backupPolicy"
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

<script lang="ts" setup>
import { computed, reactive, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { cloneDeep } from "lodash-es";
import { useRouter } from "vue-router";

import ArchiveBanner from "@/components/ArchiveBanner.vue";
import EnvironmentForm from "@/components/EnvironmentForm.vue";
import { environmentV1Slug, idFromSlug } from "@/utils";
import { hasFeature, pushNotification, useBackupV1Store } from "@/store";
import BBModal from "@/bbkit/BBModal.vue";
import {
  usePolicyV1Store,
  defaultBackupSchedule,
  defaultApprovalStrategy,
  getDefaultBackupPlanPolicy,
  getDefaultDeploymentApprovalPolicy,
} from "@/store/modules/v1/policy";
import {
  Policy as PolicyV1,
  PolicyType as PolicyTypeV1,
  PolicyResourceType,
  BackupPlanSchedule,
} from "@/types/proto/v1/org_policy_service";
import {
  useEnvironmentV1Store,
  defaultEnvironmentTier,
} from "@/store/modules/v1/environment";
import {
  Environment,
  EnvironmentTier,
} from "@/types/proto/v1/environment_service";
import { State } from "@/types/proto/v1/common";

interface LocalState {
  environment: Environment;
  showArchiveModal: boolean;
  approvalPolicy?: PolicyV1;
  backupPolicy?: PolicyV1;
  environmentTier?: EnvironmentTier;
  missingRequiredFeature?:
    | "bb.feature.approval-policy"
    | "bb.feature.backup-policy"
    | "bb.feature.environment-tier-policy";
  showDisableAutoBackupModal: boolean;
}

const props = defineProps({
  environmentSlug: {
    required: true,
    type: String,
  },
});

const emit = defineEmits(["archive"]);

const environmentV1Store = useEnvironmentV1Store();
const policyV1Store = usePolicyV1Store();
const backupStore = useBackupV1Store();
const router = useRouter();
const { t } = useI18n();

const state = reactive<LocalState>({
  environment: environmentV1Store.getEnvironmentByUID(
    String(idFromSlug(props.environmentSlug))
  ),
  showArchiveModal: false,
  showDisableAutoBackupModal: false,
});

const preparePolicy = () => {
  policyV1Store
    .getOrFetchPolicyByParentAndType({
      parentPath: state.environment.name,
      policyType: PolicyTypeV1.DEPLOYMENT_APPROVAL,
    })
    .then((policy) => {
      state.approvalPolicy =
        policy ||
        getDefaultDeploymentApprovalPolicy(
          state.environment.name,
          PolicyResourceType.ENVIRONMENT
        );
    });

  policyV1Store
    .getOrFetchPolicyByParentAndType({
      parentPath: state.environment.name,
      policyType: PolicyTypeV1.BACKUP_PLAN,
    })
    .then((policy) => {
      state.backupPolicy =
        policy ||
        getDefaultBackupPlanPolicy(
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

      const disallowed = environment.tier === EnvironmentTier.PROTECTED;
      if (disallowed) {
        return policyV1Store.upsertPolicy({
          parentPath: environment.name,
          updateMask: ["payload", "inherit_from_parent"],
          policy: {
            type: PolicyTypeV1.ACCESS_CONTROL,
            inheritFromParent: true,
            accessControlPolicy: {
              disallowRules: [{ fullDatabase: true }],
            },
          },
        });
      } else {
        policyV1Store
          .getOrFetchPolicyByParentAndType({
            parentPath: environment.name,
            policyType: PolicyTypeV1.ACCESS_CONTROL,
          })
          .then((existingPolicy) => {
            if (existingPolicy) {
              policyV1Store.deletePolicy(existingPolicy.name);
            }
          });
      }
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
    router.replace(`/environment/${environmentV1Slug(environment)}`);
  });
};

const doRestore = (environment: Environment) => {
  environmentV1Store
    .undeleteEnvironment(environment.name)
    .then((environment) => {
      assignEnvironment(environment);
      router.replace(`/environment#${environment.uid}`);
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
    parentPath: environment.name,
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
      state.backupPolicy?.backupPlanPolicy?.schedule == BackupPlanSchedule.UNSET
    ) {
      // Changing backup policy from "DAILY"|"WEEKLY" to "UNSET"
      state.showDisableAutoBackupModal = true;
    }
  }
};

const disableAutoBackupContent = computed(() => {
  return t("environment.disable-auto-backup.content");
});

const disableEnvironmentAutoBackup = async () => {
  await backupStore.upsertEnvironmentBackupSetting({
    name: `${state.environment.name}/backupSetting`,
    enabled: false,
  });
  success();
  state.showDisableAutoBackupModal = false;
};
</script>
