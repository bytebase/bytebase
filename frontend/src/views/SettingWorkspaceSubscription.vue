<template>
  <div class="w-full mx-auto">
    <div
      class="w-full grid grid-cols-2 gap-6 lg:grid-cols-3 3xl:grid-cols-4 my-4"
    >
      <div class="flex flex-col text-left">
        <div class="flex text-main">
          {{ $t("subscription.current") }}
          <span
            v-if="isExpired"
            class="ml-2 inline-flex items-center px-3 py-0.5 rounded-full text-base font-sm bg-red-100 text-red-800 h-6"
          >
            {{ $t("subscription.expired") }}
          </span>
          <span
            v-else-if="isTrialing"
            class="ml-2 inline-flex items-center px-3 py-0.5 rounded-full text-base font-sm bg-indigo-100 text-indigo-800 h-6"
          >
            {{ $t("subscription.trialing") }}
          </span>
        </div>
        <div class="text-indigo-600 mt-1 text-3xl lg:text-4xl">
          {{ currentPlan }}
        </div>
      </div>
      <div v-if="!subscriptionStore.isFreePlan" class="flex flex-col text-left">
        <div class="text-main">
          {{ $t("subscription.expires-at") }}
        </div>
        <dd class="mt-1 text-3xl lg:text-4xl">{{ expireAt || "N/A" }}</dd>
      </div>
      <div
        v-if="subscriptionStore.showTrial && allowEdit"
        class="flex flex-col text-left"
      >
        <div class="text-main">
          {{ $t("subscription.try-for-free") }}
        </div>
        <div class="mt-1">
          <NButton
            type="primary"
            size="large"
            @click="state.showTrialModal = true"
          >
            {{
              $t("subscription.enterprise-free-trial", {
                days: subscriptionStore.trialingDays,
              })
            }}
          </NButton>
        </div>
      </div>
      <div
        v-if="
          subscriptionStore.isTrialing &&
          subscriptionStore.currentPlan == PlanType.ENTERPRISE
        "
        class="flex flex-col text-left"
      >
        <div class="text-main">
          {{ $t("subscription.inquire-enterprise-plan") }}
        </div>
        <div class="mt-1 ml-auto">
          <NButton type="primary" size="large" @click="inquireEnterprise">
            {{ $t("subscription.contact-us") }}
          </NButton>
        </div>
      </div>
      <WorkspaceInstanceLicenseStats v-if="allowManageInstanceLicenses" />
      <div class="flex flex-col text-left">
        <div class="text-main">
          {{ $t("subscription.instance-assignment.used-and-total-user") }}
        </div>
        <div class="mt-1 text-4xl flex items-center gap-2">
          {{ activeUserCountWithoutBot }}
          <span class="font-mono text-gray-500">/</span>
          {{ userLimit }}
        </div>
      </div>
    </div>
    <NDivider />
    <div>
      <label class="flex items-center gap-x-2">
        <span class="text-main">
          {{ $t("settings.general.workspace.id") }}
        </span>
      </label>
      <div class="mb-3 text-sm text-gray-400">
        {{ $t("settings.general.workspace.id-description") }}
      </div>
      <div class="mb-4 flex items-center space-x-2">
        <NInput
          ref="workspaceIdField"
          class="w-full"
          readonly
          :value="workspaceId"
          @click="selectWorkspaceId"
        />
        <CopyButton
          quaternary
          :text="false"
          :size="'small'"
          :content="workspaceId"
        />
      </div>
    </div>
    <div
      v-if="allowEdit && subscriptionStore.isSelfHostLicense"
      class="w-full mt-4 flex flex-col"
    >
      <label class="flex items-center gap-x-2">
        <span class="text-main">
          {{ $t("subscription.upload-license") }}
        </span>
      </label>
      <div class="mb-3 text-sm text-gray-400">
        {{ $t("subscription.description") }}
        {{ $t("subscription.plan-compare") }}
        <LearnMoreLink url="https://www.bytebase.com/pricing?source=console" />
        <span v-if="subscriptionStore.showTrial" class="ml-1">
          <span class="text-accent cursor-pointer" @click="openTrialModal">
            {{ $t("subscription.plan.try") }}
          </span>
        </span>
      </div>

      <NInput
        v-model:value="state.license"
        type="textarea"
        :placeholder="$t('common.sensitive-placeholder')"
      />
      <div class="ml-auto mt-3">
        <NButton
          type="primary"
          class="capitalize"
          :disabled="disabled"
          @click="uploadLicense"
        >
          {{ $t("subscription.upload-license") }}
        </NButton>
      </div>
    </div>
    <TrialModal
      v-if="state.showTrialModal"
      @cancel="state.showTrialModal = false"
    />
  </div>

  <WeChatQRModal
    v-if="state.showQRCodeModal"
    :title="$t('subscription.inquire-enterprise-plan')"
    @close="state.showQRCodeModal = false"
  />
</template>

<script lang="ts" setup>
import { NButton, NDivider, NInput } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import TrialModal from "@/components/TrialModal.vue";
import WeChatQRModal from "@/components/WeChatQRModal.vue";
import WorkspaceInstanceLicenseStats from "@/components/WorkspaceInstanceLicenseStats.vue";
import { CopyButton } from "@/components/v2";
import { useLanguage } from "@/composables/useLanguage";
import {
  pushNotification,
  useActuatorV1Store,
  useSettingV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";
import { Setting_SettingName } from "@/types/proto/v1/setting_service";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  loading: boolean;
  license: string;
  showQRCodeModal: boolean;
  showTrialModal: boolean;
}

const props = defineProps<{
  allowEdit: boolean;
}>();

const { t } = useI18n();
const { locale } = useLanguage();
const subscriptionStore = useSubscriptionV1Store();
const settingV1Store = useSettingV1Store();
const actuatorStore = useActuatorV1Store();

const state = reactive<LocalState>({
  loading: false,
  license: "",
  showTrialModal: false,
  showQRCodeModal: false,
});

const disabled = computed((): boolean => {
  return state.loading || !state.license;
});

const activeUserCountWithoutBot = computed(() =>
  actuatorStore.getActiveUserCount({ includeBot: false })
);

const userLimit = computed((): string => {
  if (subscriptionStore.userCountLimit === Number.MAX_VALUE) {
    return t("subscription.unlimited");
  }
  return `${subscriptionStore.userCountLimit}`;
});

const workspaceIdField = ref<HTMLInputElement | null>(null);

const workspaceId = computed(() => {
  return (
    settingV1Store.getSettingByName(Setting_SettingName.WORKSPACE_ID)?.value
      ?.stringValue ?? ""
  );
});

const selectWorkspaceId = () => {
  workspaceIdField.value?.select();
};

const uploadLicense = async () => {
  if (disabled.value) return;
  state.loading = true;

  try {
    await subscriptionStore.patchSubscription(state.license);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("subscription.update.success.title"),
      description: t("subscription.update.success.description"),
    });
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("subscription.update.failure.title"),
      description: t("subscription.update.failure.description"),
    });
  } finally {
    state.loading = false;
    state.license = "";
  }
};

const { expireAt, isTrialing, isExpired } = storeToRefs(subscriptionStore);

const allowManageInstanceLicenses = computed(() => {
  return props.allowEdit && hasWorkspacePermissionV2("bb.instances.list");
});

const currentPlan = computed((): string => {
  const plan = subscriptionStore.currentPlan;
  switch (plan) {
    case PlanType.TEAM:
      return t("subscription.plan.team.title");
    case PlanType.ENTERPRISE:
      return t("subscription.plan.enterprise.title");
    default:
      return t("subscription.plan.free.title");
  }
});

const openTrialModal = () => {
  state.showTrialModal = true;
};

const inquireEnterprise = () => {
  if (locale.value === "zh-CN") {
    state.showQRCodeModal = true;
  } else {
    window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
  }
};
</script>
