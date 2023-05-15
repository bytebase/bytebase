<template>
  <div v-if="showBanner" class="bg-warning">
    <div
      class="mx-auto py-3 px-3 w-full flex flex-row items-center justify-between flex-wrap"
    >
      <div class="flex flex-row items-center">
        <heroicons-outline:megaphone class="w-8 h-auto text-gray-800 mr-1" />
        <i18n-t
          tag="p"
          keypath="subscription.overuse-warning"
          class="text-white text-lg"
        >
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
      <button class="bg-white btn btn-normal" @click="gotoSubscriptionPage">
        {{ $t("subscription.button.upgrade") }}
        <heroicons-outline:sparkles class="w-5 h-auto text-accent ml-1" />
      </button>
    </div>
  </div>

  <BBModal
    v-if="state.showModal"
    class="!w-128"
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
  useSettingStore,
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
const settingStore = useSettingStore();
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
  const brandingSetting = settingStore.getSettingByName("bb.branding.logo");
  if (brandingSetting && brandingSetting.value) {
    list.push("bb.feature.branding");
  }
  const watermarkSetting = settingStore.getSettingByName(
    "bb.workspace.watermark"
  );
  if (watermarkSetting && watermarkSetting.value === "1") {
    list.push("bb.feature.watermark");
  }
  if (settingStore.workspaceSetting?.disallowSignup ?? false) {
    list.push("bb.feature.disallow-signup");
  }
  if (settingStore.workspaceSetting?.require2fa ?? false) {
    list.push("bb.feature.2fa");
  }
  const openAIKeySetting = settingStore.getSettingByName(
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
    settingStore.fetchSetting(),
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
