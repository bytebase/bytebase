<template>
  <div v-if="showBanner" class="bg-gray-200 overflow-clip">
    <div class="w-full h-10 scroll-animation">
      <div
        class="mx-auto py-1 px-3 w-full flex flex-row items-center justify-center flex-wrap"
      >
        <div class="flex flex-row items-center">
          <heroicons-outline:exclamation-circle
            class="w-5 h-auto text-gray-800 mr-1"
          />
          <i18n-t tag="p" keypath="subscription.overuse-warning">
            <template #neededPlan>
              <span
                class="underline cursor-pointer hover:opacity-60"
                @click="state.showModal = true"
                >{{
                  t("subscription.plan-features", {
                    plan: t(
                      `subscription.plan.${planTypeToString(neededPlan)}.title`
                    ),
                  })
                }}</span
              >
            </template>
            <template #currentPlan>
              {{ currentPlan }}
            </template>
          </i18n-t>
        </div>
        <button
          class="btn btn-normal btn-small flex flex-row justify-center items-center ml-2 !py-1 px-2"
          @click="gotoSubscriptionPage"
        >
          {{ $t("subscription.button.upgrade") }}
          <heroicons-outline:sparkles class="w-4 h-auto text-accent ml-1" />
        </button>
      </div>
    </div>
  </div>

  <BBModal
    v-if="state.showModal"
    class="!w-112"
    :title="$t('subscription.upgrade-now') + '?'"
    @close="state.showModal = false"
  >
    <p>
      {{ $t("subscription.overuse-modal.description", { plan: currentPlan }) }}
    </p>
    <div class="pl-4 my-2">
      <ul class="list-disc list-inside">
        <li v-for="feature in overUsedFeatureList" :key="feature">
          {{
            $t(`subscription.features.${feature.split(".").join("-")}.title`)
          }}
        </li>
      </ul>
    </div>
    <button
      class="mt-3 mb-4 w-full btn btn-primary"
      @click="gotoSubscriptionPage"
    >
      <span class="w-full text-center">{{
        $t("subscription.upgrade-now")
      }}</span>
    </button>
  </BBModal>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import {
  useIdentityProviderStore,
  useInstanceV1Store,
  useSubscriptionV1Store,
  usePolicyV1Store,
  useSettingV1Store,
  defaultBackupSchedule,
  defaultApprovalStrategy,
  useDBGroupStore,
  useDatabaseV1Store,
  useDatabaseSecretStore,
} from "@/store";
import { onMounted } from "vue";
import { computed } from "vue";
import { FeatureType, planTypeToString } from "@/types";
import { useEnvironmentV1Store } from "@/store/modules/v1/environment";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import { useI18n } from "vue-i18n";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { useRouter } from "vue-router";
import { PolicyType } from "@/types/proto/v1/org_policy_service";

interface LocalState {
  showModal: boolean;
}

const { t } = useI18n();
const router = useRouter();
const subscriptionStore = useSubscriptionV1Store();
const state = reactive<LocalState>({
  showModal: false,
});

const idpStore = useIdentityProviderStore();
const settingV1Store = useSettingV1Store();
const instanceStore = useInstanceV1Store();
const environmentV1Store = useEnvironmentV1Store();
const policyV1Store = usePolicyV1Store();
const dbV1Store = useDatabaseV1Store();
const dbSecretStore = useDatabaseSecretStore();
const dbGroupStore = useDBGroupStore();

const usedFeatureList = computed(() => {
  const set: Set<FeatureType> = new Set();

  if (idpStore.identityProviderList.length > 0) {
    set.add("bb.feature.sso");
  }

  // setting
  if (settingV1Store.brandingLogo) {
    set.add("bb.feature.branding");
  }
  const watermarkSetting = settingV1Store.getSettingByName(
    "bb.workspace.watermark"
  );
  if (watermarkSetting?.value?.stringValue === "1") {
    set.add("bb.feature.watermark");
  }
  if (settingV1Store.workspaceProfileSetting?.disallowSignup ?? false) {
    set.add("bb.feature.disallow-signup");
  }
  if (settingV1Store.workspaceProfileSetting?.require2fa ?? false) {
    set.add("bb.feature.2fa");
  }
  const openAIKeySetting = settingV1Store.getSettingByName(
    "bb.plugin.openai.key"
  );
  if (openAIKeySetting?.value?.stringValue) {
    set.add("bb.feature.plugin.openai");
  }
  const imSetting = settingV1Store.getSettingByName("bb.app.im");
  if (imSetting?.value?.appImSettingValue?.appId) {
    set.add("bb.feature.im.approval");
  }

  // environment
  for (const environment of environmentV1Store.environmentList) {
    if (environment.tier === EnvironmentTier.PROTECTED) {
      set.add("bb.feature.environment-tier-policy");
    }
    const backupPolicy = policyV1Store.getPolicyByParentAndType({
      parentPath: environment.name,
      policyType: PolicyType.BACKUP_PLAN,
    });
    if (
      backupPolicy?.backupPlanPolicy?.schedule ??
      defaultBackupSchedule !== defaultBackupSchedule
    ) {
      set.add("bb.feature.backup-policy");
    }

    const approvalPolicy = policyV1Store.getPolicyByParentAndType({
      parentPath: environment.name,
      policyType: PolicyType.DEPLOYMENT_APPROVAL,
    });
    if (
      approvalPolicy?.deploymentApprovalPolicy?.defaultStrategy ??
      defaultApprovalStrategy !== defaultApprovalStrategy
    ) {
      set.add("bb.feature.approval-policy");
    }
  }

  // database
  for (const databse of dbV1Store.databaseList) {
    const list = dbSecretStore.getSecretListByDatabase(databse.name);
    if (list.length > 0) {
      set.add("bb.feature.encrypted-secrets");
      break;
    }
  }
  if (dbGroupStore.getAllDatabaseGroupList().length > 0) {
    set.add("bb.feature.database-grouping");
  }

  return [...set.values()];
});

const overUsedFeatureList = computed(() => {
  const currentPlan = subscriptionStore.currentPlan;
  return usedFeatureList.value.filter((feature) => {
    const requiredPlan = subscriptionStore.getMinimumRequiredPlan(feature);
    return requiredPlan > currentPlan;
  });
});

const neededPlan = computed(() => {
  let plan = PlanType.FREE;

  for (const feature of usedFeatureList.value) {
    const requiredPlan = subscriptionStore.getMinimumRequiredPlan(feature);
    if (requiredPlan > plan) {
      plan = requiredPlan;
    }
  }

  return plan;
});

const showBanner = computed(() => {
  return (
    overUsedFeatureList.value.length > 0 &&
    neededPlan.value > subscriptionStore.currentPlan
  );
});

const currentPlan = computed(() => {
  return t(
    `subscription.plan.${planTypeToString(subscriptionStore.currentPlan)}.title`
  );
});

onMounted(() => {
  if (subscriptionStore.currentPlan === PlanType.ENTERPRISE) {
    return;
  }

  Promise.all([
    idpStore.fetchIdentityProviderList(),
    settingV1Store.fetchSettingList(),
    environmentV1Store.fetchEnvironments().then((list) => {
      return list.map((env) => {
        Promise.all([
          policyV1Store.getOrFetchPolicyByParentAndType({
            parentPath: env.name,
            policyType: PolicyType.DEPLOYMENT_APPROVAL,
          }),
          policyV1Store.getOrFetchPolicyByParentAndType({
            parentPath: env.name,
            policyType: PolicyType.BACKUP_PLAN,
          }),
        ]);
      });
    }),
    instanceStore.fetchInstanceList(),
    dbV1Store
      .fetchDatabaseList({
        parent: "instances/-",
      })
      .then((list) => {
        return list.map((db) => dbSecretStore.fetchSecretList(db.name));
      }),
    dbGroupStore.fetchAllDatabaseGroupList(),
  ]).catch(() => {
    // ignore
  });
});

const gotoSubscriptionPage = () => {
  state.showModal = false;
  return router.push({ name: "setting.workspace.subscription" });
};
</script>

<style>
.scroll-animation {
  display: inline-block;
  animation: scroll 4s ease-in-out infinite;
}

@keyframes scroll {
  0% {
    transform: translateY(100%);
  }
  25% {
    transform: translateY(0);
  }
  80% {
    transform: translateY(0);
  }
  100% {
    transform: translateY(-100%);
  }
}
</style>
