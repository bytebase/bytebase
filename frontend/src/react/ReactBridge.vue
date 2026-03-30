<template>
  <div ref="container" />
  <!-- Subscription page Vue modals -->
  <template v-if="page === 'SubscriptionPage'">
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
</template>

<script lang="ts" setup>
import { storeToRefs } from "pinia";
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import InstanceAssignment from "@/components/InstanceAssignment.vue";
import WeChatQRModal from "@/components/WeChatQRModal.vue";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import {
  getWorkspaceId,
  pushNotification,
  useActuatorV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";
import { PlanType } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = withDefaults(
  defineProps<{
    page: string;
    allowEdit?: boolean;
  }>(),
  { allowEdit: false }
);

const { t, locale } = useI18n();
const router = useRouter();
const actuatorStore = useActuatorV1Store();
const subscriptionStore = useSubscriptionV1Store();
const { expireAt, isTrialing, isExpired } = storeToRefs(subscriptionStore);

const container = ref<HTMLElement>();
// biome-ignore lint/suspicious/noExplicitAny: React Root type from dynamic import
let root: any = null; // eslint-disable-line @typescript-eslint/no-explicit-any

// --- Subscription-specific Vue state ---
const showQRCodeModal = ref(false);
const showInstanceAssignmentDrawer = ref(false);

const handleRequireEnterprise = () => {
  if (locale.value === "zh-CN") {
    showQRCodeModal.value = true;
  } else {
    window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
  }
};

// --- Props builders per page ---

function buildSubscriptionProps() {
  let planType: "FREE" | "TEAM" | "ENTERPRISE" = "FREE";
  let currentPlan = t("subscription.plan.free.title");
  if (subscriptionStore.currentPlan === PlanType.TEAM) {
    planType = "TEAM";
    currentPlan = t("subscription.plan.team.title");
  } else if (subscriptionStore.currentPlan === PlanType.ENTERPRISE) {
    planType = "ENTERPRISE";
    currentPlan = t("subscription.plan.enterprise.title");
  }
  return {
    data: {
      currentPlan,
      planType,
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
    },
    allowEdit: props.allowEdit,
    allowManageInstanceLicenses:
      props.allowEdit && hasWorkspacePermissionV2("bb.instances.list"),
    onUploadLicense: async (license: string): Promise<boolean> => {
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
    },
    onRequireEnterprise: handleRequireEnterprise,
    onManageInstanceLicenses: () => {
      showInstanceAssignmentDrawer.value = true;
    },
  };
}

function buildMCPProps() {
  return {
    externalUrl: actuatorStore.serverInfo?.externalUrl ?? "",
    needConfigureExternalUrl: actuatorStore.needConfigureExternalUrl,
    canConfigureExternalUrl: hasWorkspacePermissionV2(
      "bb.settings.setWorkspaceProfile"
    ),
    onConfigureExternalUrl: () => {
      router.push({ name: SETTING_ROUTE_WORKSPACE_GENERAL });
    },
  };
}

const propsBuilders: Record<string, () => Record<string, unknown>> = {
  SubscriptionPage: buildSubscriptionProps,
  MCPPage: buildMCPProps,
};

function buildPageProps() {
  const builder = propsBuilders[props.page];
  if (!builder) throw new Error(`Unknown React page: ${props.page}`);
  return builder();
}

// --- Reactive dependencies to watch per page ---

const watchDeps = computed(() => {
  if (props.page === "SubscriptionPage") {
    return [
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
    ];
  }
  if (props.page === "MCPPage") {
    return [
      actuatorStore.serverInfo?.externalUrl,
      actuatorStore.needConfigureExternalUrl,
      locale.value,
    ];
  }
  return [locale.value];
});

// --- Mount / update / unmount ---

async function renderReact() {
  if (!container.value) return;
  const [{ mountReactPage, updateReactPage }, i18nModule] = await Promise.all([
    import("./mount"),
    import("./i18n"),
  ]);
  if (i18nModule.default.language !== locale.value) {
    await i18nModule.default.changeLanguage(locale.value);
  }
  const pageProps = buildPageProps();
  if (!root) {
    root = await mountReactPage(container.value, props.page, pageProps);
  } else {
    await updateReactPage(root, props.page, pageProps);
  }
}

onMounted(() => {
  renderReact();
});

watch(watchDeps, () => {
  renderReact();
});

onUnmounted(() => {
  root?.unmount();
  root = null;
});
</script>
