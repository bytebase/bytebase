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
        <li v-for="item in overUsedFeatureList" :key="item.feature">
          {{
            $t(
              `subscription.features.${item.feature.split(".").join("-")}.title`
            )
          }}
          ({{
            $t(
              `subscription.plan.${planTypeToString(item.requiredPlan)}.title`
            )
          }})
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
import { reactive, watch, ref } from "vue";
import { onMounted } from "vue";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
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
  useActuatorV1Store,
} from "@/store";
import { useEnvironmentV1Store } from "@/store/modules/v1/environment";
import { FeatureType, planTypeToString, refreshTokenDurationInHours } from "@/types";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import { PlanType } from "@/types/proto/v1/subscription_service";

interface LocalState {
  showModal: boolean;
  ready: boolean;
}

const { t } = useI18n();
const router = useRouter();
const subscriptionStore = useSubscriptionV1Store();
const state = reactive<LocalState>({
  showModal: false,
  ready: false,
});

const actuatorStore = useActuatorV1Store();
const idpStore = useIdentityProviderStore();
const settingV1Store = useSettingV1Store();
const instanceStore = useInstanceV1Store();
const environmentV1Store = useEnvironmentV1Store();
const policyV1Store = usePolicyV1Store();
const dbV1Store = useDatabaseV1Store();
const dbSecretStore = useDatabaseSecretStore();
const dbGroupStore = useDBGroupStore();

const usedFeatureList = ref<FeatureType[]>([]);

watch(
  () => state.ready,
  async (ready) => {
    if (!ready) {
      return;
    }
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
    if (
      settingV1Store.workspaceProfileSetting?.refreshTokenDuration?.seconds != refreshTokenDurationInHours
    ) {
      set.add("bb.feature.secure-token");
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

    for (const environment of environmentV1Store.environmentList) {
      if (environment.tier === EnvironmentTier.PROTECTED) {
        set.add("bb.feature.environment-tier-policy");
      }
      if (!set.has("bb.feature.backup-policy")) {
        const backupPolicy =
          await policyV1Store.getOrFetchPolicyByParentAndType({
            parentPath: environment.name,
            policyType: PolicyType.BACKUP_PLAN,
          });
        if (
          backupPolicy?.backupPlanPolicy?.schedule &&
          backupPolicy?.backupPlanPolicy?.schedule !== defaultBackupSchedule
        ) {
          set.add("bb.feature.backup-policy");
        }
      }

      if (!set.has("bb.feature.approval-policy")) {
        const approvalPolicy =
          await policyV1Store.getOrFetchPolicyByParentAndType({
            parentPath: environment.name,
            policyType: PolicyType.DEPLOYMENT_APPROVAL,
          });
        if (
          approvalPolicy?.deploymentApprovalPolicy?.defaultStrategy &&
          approvalPolicy?.deploymentApprovalPolicy?.defaultStrategy !==
            defaultApprovalStrategy
        ) {
          set.add("bb.feature.approval-policy");
        }
      }
    }

    // database
    for (const databse of dbV1Store.databaseList) {
      const list = await dbSecretStore.fetchSecretList(databse.name);
      if (list.length > 0) {
        set.add("bb.feature.encrypted-secrets");
        break;
      }
    }
    if (dbGroupStore.getAllDatabaseGroupList().length > 0) {
      set.add("bb.feature.database-grouping");
    }

    usedFeatureList.value = [...set.values()];
  }
);

const overUsedFeatureList = computed(() => {
  const currentPlan = subscriptionStore.currentPlan;
  const resp = [];
  for (const feature of usedFeatureList.value) {
    const requiredPlan = subscriptionStore.getMinimumRequiredPlan(feature);
    if (requiredPlan > currentPlan) {
      resp.push({
        feature,
        requiredPlan,
      });
    }
  }
  return resp;
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
    // Do not show banner in demo mode
    actuatorStore.serverInfo?.demoName == "" &&
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
  if (subscriptionStore.currentPlan !== PlanType.FREE) {
    return;
  }

  Promise.all([
    idpStore.fetchIdentityProviderList(),
    settingV1Store.fetchSettingList(),
    environmentV1Store.fetchEnvironments(),
    instanceStore.fetchInstanceList(),
    dbGroupStore.fetchAllDatabaseGroupList(),
  ])
    .catch(() => {
      // ignore
    })
    .finally(() => {
      state.ready = true;
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
