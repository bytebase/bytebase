<template>
  <div ref="container" />
  <WeChatQRModal
    v-if="showQRCodeModal"
    :title="$t('subscription.request-with-qr')"
    @close="showQRCodeModal = false"
  />
  <InstanceAssignment
    :show="showInstanceAssignmentDrawer"
    @dismiss="showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import InstanceAssignment from "@/components/InstanceAssignment.vue";
import WeChatQRModal from "@/components/WeChatQRModal.vue";
import {
  getWorkspaceId,
  pushNotification,
  useActuatorV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = defineProps<{
  allowEdit: boolean;
}>();

const { t, locale } = useI18n();
const subscriptionStore = useSubscriptionV1Store();
const actuatorStore = useActuatorV1Store();

const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

const { expireAt, isTrialing, isExpired } = storeToRefs(subscriptionStore);

const showQRCodeModal = ref(false);
const showInstanceAssignmentDrawer = ref(false);

const handleRequireEnterprise = () => {
  if (locale.value === "zh-CN") {
    showQRCodeModal.value = true;
  } else {
    window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
  }
};

function getPlanType(): "FREE" | "TEAM" | "ENTERPRISE" {
  switch (subscriptionStore.currentPlan) {
    case PlanType.TEAM:
      return "TEAM";
    case PlanType.ENTERPRISE:
      return "ENTERPRISE";
    default:
      return "FREE";
  }
}

function getCurrentPlanLabel(): string {
  switch (subscriptionStore.currentPlan) {
    case PlanType.TEAM:
      return t("subscription.plan.team.title");
    case PlanType.ENTERPRISE:
      return t("subscription.plan.enterprise.title");
    default:
      return t("subscription.plan.free.title");
  }
}

function buildData() {
  return {
    currentPlan: getCurrentPlanLabel(),
    planType: getPlanType(),
    isFreePlan: subscriptionStore.isFreePlan,
    isTrialing: isTrialing.value,
    isExpired: isExpired.value,
    isSelfHostLicense: subscriptionStore.isSelfHostLicense,
    showTrial: subscriptionStore.showTrial,
    trialingDays: subscriptionStore.trialingDays,
    expireAt: expireAt.value,
    instanceCountLimit: subscriptionStore.instanceCountLimit,
    instanceLicenseCount: subscriptionStore.instanceLicenseCount,
    userCountLimit: subscriptionStore.userCountLimit,
    activeUserCount: actuatorStore.activeUserCount,
    activatedInstanceCount: actuatorStore.activatedInstanceCount,
    workspaceId: getWorkspaceId(actuatorStore.workspaceResourceName),
  };
}

async function handleUploadLicense(license: string): Promise<boolean> {
  try {
    await subscriptionStore.patchSubscription(license);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("subscription.update.success.title"),
      description: t("subscription.update.success.description"),
    });
    return true;
  } catch {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("subscription.update.failure.title"),
      description: t("subscription.update.failure.description"),
    });
    return false;
  }
}

async function renderReact() {
  if (!container.value) return;
  const [{ mountSubscriptionPage, updateSubscriptionPage }, i18nModule] =
    await Promise.all([import("./mount"), import("./i18n")]);
  // Sync Vue locale to React i18next
  if (i18nModule.default.language !== locale.value) {
    await i18nModule.default.changeLanguage(locale.value);
  }
  const opts = {
    data: buildData(),
    allowEdit: props.allowEdit,
    allowManageInstanceLicenses:
      props.allowEdit && hasWorkspacePermissionV2("bb.instances.list"),
    onUploadLicense: handleUploadLicense,
    onRequireEnterprise: handleRequireEnterprise,
    onManageInstanceLicenses: () => {
      showInstanceAssignmentDrawer.value = true;
    },
  };
  if (!root) {
    root = await mountSubscriptionPage(container.value, opts);
  } else {
    await updateSubscriptionPage(root, opts);
  }
}

onMounted(() => {
  renderReact();
});

// Re-render when reactive data changes
watch(
  () => [
    subscriptionStore.currentPlan,
    subscriptionStore.isFreePlan,
    isTrialing.value,
    isExpired.value,
    expireAt.value,
    subscriptionStore.instanceCountLimit,
    subscriptionStore.instanceLicenseCount,
    subscriptionStore.userCountLimit,
    actuatorStore.activeUserCount,
    actuatorStore.activatedInstanceCount,
    props.allowEdit,
    locale.value,
  ],
  () => {
    renderReact();
  }
);

onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
