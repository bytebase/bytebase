<template>
  <div v-if="showBanner" class="bg-gray-200 overflow-clip">
    <div class="w-full scroll-animation">
      <div
        class="mx-auto py-1 px-3 w-full flex flex-row items-center justify-center flex-wrap"
      >
        <div class="flex flex-row items-center">
          <heroicons-outline:megaphone class="w-5 h-auto text-gray-800 mr-1" />
          <i18n-t tag="p" keypath="subscription.overuse-warning">
            <template #neededPlan>
              <span
                class="underline cursor-pointer hover:opacity-80"
                @click="state.showModal = true"
                >{{ neededPlan }}</span
              >
            </template>
            <template #currentPlan>
              {{ currentPlan }}
            </template>
          </i18n-t>
        </div>
        <button
          class="bg-white btn btn-normal btn-small flex flex-row justify-center items-center ml-2 !py-1"
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
        <li v-for="feature in overusedFeatureList" :key="feature">
          {{ feature }}
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
  useInstanceStore,
  useSubscriptionStore,
} from "@/store";
import { onMounted } from "vue";
import { computed } from "vue";
import { FeatureType, planTypeToString } from "@/types";
import { useEnvironmentV1Store } from "@/store/modules/v1/environment";
import { EnvironmentTier } from "@/types/proto/v1/environment_service";
import { useI18n } from "vue-i18n";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { useRouter } from "vue-router";
import { useSettingV1Store } from "@/store/modules/v1/setting";

interface LocalState {
  showModal: boolean;
}

const { t } = useI18n();
const router = useRouter();
const subscriptionStore = useSubscriptionStore();
const state = reactive<LocalState>({
  showModal: false,
});

const idpStore = useIdentityProviderStore();
const settingV1Store = useSettingV1Store();
const instanceStore = useInstanceStore();
const environmentV1Store = useEnvironmentV1Store();

const showBanner = computed(() => {
  return (
    subscriptionStore.currentPlan !== PlanType.ENTERPRISE &&
    overusedFeatureList.value.length > 0
  );
});

const neededPlan = computed(() => {
  let plan = PlanType.TEAM;
  if (overusedEnterprisePlanFeatureList.value.length > 0) {
    plan = PlanType.ENTERPRISE;
  }
  return t("subscription.plan-features", {
    plan: t(`subscription.plan.${planTypeToString(plan)}.title`),
  });
});

const currentPlan = computed(() => {
  return t(
    `subscription.plan.${planTypeToString(subscriptionStore.currentPlan)}.title`
  );
});

const overusedEnterprisePlanFeatureList = computed(() => {
  if (subscriptionStore.currentPlan === PlanType.ENTERPRISE) {
    return [];
  }

  const list: FeatureType[] = [];

  if (idpStore.identityProviderList.length > 0) {
    list.push("bb.feature.sso");
  }
  if (settingV1Store.brandingLogo) {
    list.push("bb.feature.branding");
  }
  const watermarkSetting = settingV1Store.getSettingByName(
    "bb.workspace.watermark"
  );
  if (watermarkSetting && watermarkSetting.value?.stringValue === "1") {
    list.push("bb.feature.watermark");
  }
  if (settingV1Store.workspaceProfileSetting?.disallowSignup ?? false) {
    list.push("bb.feature.disallow-signup");
  }
  if (settingV1Store.workspaceProfileSetting?.require2fa ?? false) {
    list.push("bb.feature.2fa");
  }
  const openAIKeySetting = settingV1Store.getSettingByName(
    "bb.plugin.openai.key"
  );
  if (openAIKeySetting && openAIKeySetting.value) {
    list.push("bb.feature.plugin.openai");
  }
  for (const environment of environmentV1Store.environmentList) {
    if (environment.tier === EnvironmentTier.PROTECTED) {
      list.push("bb.feature.environment-tier-policy");
      break;
    }
  }

  return list;
});

const overusedFeatureList = computed(() => {
  const list: string[] = [];
  for (const feature of overusedEnterprisePlanFeatureList.value) {
    list.push(t(`subscription.features.${feature.split(".").join("-")}.title`));
  }
  if (instanceStore.instanceById.size > subscriptionStore.instanceCount) {
    list.push(t("subscription.overuse-modal.instance-count-exceeds"));
  }
  return list;
});

onMounted(() => {
  if (subscriptionStore.currentPlan === PlanType.ENTERPRISE) {
    return;
  }

  Promise.all([
    idpStore.fetchIdentityProviderList(),
    settingV1Store.fetchSettingList(),
    environmentV1Store.fetchEnvironments(),
    instanceStore.fetchInstanceList(),
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
