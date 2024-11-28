<template>
  <div class="mx-auto">
    <div v-if="subscriptionStore.isSelfHostLicense" class="textinfolabel mb-4">
      {{ $t("subscription.description") }}
      <a
        class="text-accent"
        :href="subscriptionStore.purchaseLicenseUrl"
        target="__blank"
      >
        {{ $t("subscription.purchase-license") }}
      </a>
      <span v-if="subscriptionStore.canTrial" class="ml-1">
        {{ $t("common.or") }}
        <span class="text-accent cursor-pointer" @click="openTrialModal">
          {{ $t("subscription.plan.try") }}
        </span>
      </span>
    </div>
    <div>
      <label class="flex items-center gap-x-2">
        <span class="text-main">
          {{ $t("settings.general.workspace.id") }}
        </span>
      </label>
      <div class="mb-3 text-sm text-gray-400">
        {{ $t("settings.general.workspace.id-description") }}
      </div>
      <div class="mb-4 flex space-x-2">
        <NInput
          ref="workspaceIdField"
          class="mb-4 w-full"
          readonly
          :value="workspaceId"
          @click="selectWorkspaceId"
        />
        <NButton
          v-if="isSupported"
          :disabled="!workspaceId"
          @click="handleCopyId"
        >
          <heroicons-outline:clipboard-document class="w-4 h-4" />
        </NButton>
      </div>
    </div>
    <NDivider />
    <dl class="text-left grid grid-cols-2 gap-x-6 my-4 xl:grid-cols-4">
      <div class="my-3">
        <dt class="flex text-main">
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
        </dt>
        <dd class="text-indigo-600 mt-1 text-4xl">
          <div>
            {{ currentPlan }}
          </div>
        </dd>
      </div>
      <WorkspaceInstanceLicenseStats v-if="allowManageInstanceLicenses" />
      <div v-if="!subscriptionStore.isFreePlan" class="my-3">
        <dt class="text-main">
          {{ $t("subscription.expires-at") }}
        </dt>
        <dd class="mt-1 text-4xl">{{ expireAt || "n/a" }}</dd>
      </div>
      <div v-if="subscriptionStore.canTrial && allowEdit" class="my-3">
        <dt class="text-main">
          {{ $t("subscription.try-for-free") }}
        </dt>

        <dd class="mt-1">
          <NButton type="primary" @click="state.showTrialModal = true">
            {{
              $t("subscription.enterprise-free-trial", {
                days: subscriptionStore.trialingDays,
              })
            }}
          </NButton>
        </dd>
      </div>
      <div
        v-if="
          subscriptionStore.isTrialing &&
          subscriptionStore.currentPlan == PlanType.ENTERPRISE
        "
        class="my-3"
      >
        <dt class="text-main">
          {{ $t("subscription.inquire-enterprise-plan") }}
        </dt>

        <dd class="mt-1 ml-auto">
          <NButton type="primary" @click="inquireEnterprise">
            {{ $t("subscription.contact-us") }}
          </NButton>
        </dd>
      </div>
    </dl>
    <div
      v-if="allowEdit && subscriptionStore.isSelfHostLicense"
      class="w-full mt-4 flex flex-col"
    >
      <NInput
        v-model:value="state.license"
        type="textarea"
        :placeholder="$t('subscription.sensitive-placeholder')"
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
    <NDivider />
    <div class="sm:flex sm:flex-col sm:align-center">
      <div class="textinfolabel">
        {{ $t("subscription.plan-compare") }}
      </div>
      <PricingTable @on-trial="openTrialModal" />
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
import { useClipboard } from "@vueuse/core";
import { NButton, NDivider, NInput } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import TrialModal from "@/components/TrialModal.vue";
import WeChatQRModal from "@/components/WeChatQRModal.vue";
import WorkspaceInstanceLicenseStats from "@/components/WorkspaceInstanceLicenseStats.vue";
import { useLanguage } from "@/composables/useLanguage";
import {
  pushNotification,
  useSubscriptionV1Store,
  useSettingV1Store,
} from "@/store";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";
import { PlanType } from "@/types/proto/v1/subscription_service";
import { hasWorkspacePermissionV2 } from "@/utils";
import PricingTable from "../components/PricingTable/";

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

const state = reactive<LocalState>({
  loading: false,
  license: "",
  showTrialModal: false,
  showQRCodeModal: false,
});

const disabled = computed((): boolean => {
  return state.loading || !state.license;
});

const workspaceIdField = ref<HTMLInputElement | null>(null);

const workspaceId = computed(() => {
  return (
    settingV1Store.getSettingByName("bb.workspace.id")?.value?.stringValue ?? ""
  );
});

const selectWorkspaceId = () => {
  workspaceIdField.value?.select();
};

const { copy: copyTextToClipboard, isSupported } = useClipboard({
  legacy: true,
});

const handleCopyId = () => {
  selectWorkspaceId();
  copyTextToClipboard(workspaceId.value).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.copied"),
    });
  });
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
