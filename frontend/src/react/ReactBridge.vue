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
import { clone, create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
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
  useSettingV1Store,
  useSubscriptionV1Store,
} from "@/store";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";
import { WebhookType } from "@/types/proto-es/v1/common_pb";
import {
  AppIMSetting_DingTalkSchema,
  AppIMSetting_FeishuSchema,
  type AppIMSetting_IMSetting,
  AppIMSetting_IMSettingSchema,
  AppIMSetting_LarkSchema,
  AppIMSetting_SlackSchema,
  AppIMSetting_TeamsSchema,
  AppIMSetting_WecomSchema,
  AppIMSettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
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

// --- IM page helpers ---

const settingStore = useSettingV1Store();

const IM_TYPES = [
  WebhookType.SLACK,
  WebhookType.FEISHU,
  WebhookType.LARK,
  WebhookType.WECOM,
  WebhookType.DINGTALK,
  WebhookType.TEAMS,
] as const;

const IM_TYPE_KEY: Record<number, string> = {
  [WebhookType.SLACK]: "SLACK",
  [WebhookType.FEISHU]: "FEISHU",
  [WebhookType.LARK]: "LARK",
  [WebhookType.WECOM]: "WECOM",
  [WebhookType.DINGTALK]: "DINGTALK",
  [WebhookType.TEAMS]: "TEAMS",
};

const IM_LABELS: Record<number, string> = {
  [WebhookType.SLACK]: "Slack",
  [WebhookType.FEISHU]: "Feishu",
  [WebhookType.LARK]: "Lark",
  [WebhookType.WECOM]: "WeCom",
  [WebhookType.DINGTALK]: "DingTalk",
  [WebhookType.TEAMS]: "Teams",
};

const IM_FIELDS: Record<number, { key: string; label: string }[]> = {
  [WebhookType.SLACK]: [{ key: "token", label: "Token" }],
  [WebhookType.FEISHU]: [
    { key: "appId", label: "App ID" },
    { key: "appSecret", label: "App Secret" },
  ],
  [WebhookType.LARK]: [
    { key: "appId", label: "App ID" },
    { key: "appSecret", label: "App Secret" },
  ],
  [WebhookType.WECOM]: [
    { key: "corpId", label: "Corp ID" },
    { key: "agentId", label: "Agent ID" },
    { key: "secret", label: "Secret" },
  ],
  [WebhookType.DINGTALK]: [
    { key: "clientId", label: "Client ID" },
    { key: "clientSecret", label: "Client Secret" },
    { key: "robotCode", label: "Robot Code" },
  ],
  [WebhookType.TEAMS]: [
    { key: "tenantId", label: "Tenant ID" },
    { key: "clientId", label: "Client ID" },
    { key: "clientSecret", label: "Client Secret" },
  ],
};

function createIMSetting(
  wt: WebhookType,
  init?: Record<string, string>
): AppIMSetting_IMSetting {
  switch (wt) {
    case WebhookType.SLACK:
      return create(AppIMSetting_IMSettingSchema, {
        type: wt,
        payload: {
          case: "slack",
          value: create(AppIMSetting_SlackSchema, init),
        },
      });
    case WebhookType.FEISHU:
      return create(AppIMSetting_IMSettingSchema, {
        type: wt,
        payload: {
          case: "feishu",
          value: create(AppIMSetting_FeishuSchema, init),
        },
      });
    case WebhookType.LARK:
      return create(AppIMSetting_IMSettingSchema, {
        type: wt,
        payload: {
          case: "lark",
          value: create(AppIMSetting_LarkSchema, init),
        },
      });
    case WebhookType.WECOM:
      return create(AppIMSetting_IMSettingSchema, {
        type: wt,
        payload: {
          case: "wecom",
          value: create(AppIMSetting_WecomSchema, init),
        },
      });
    case WebhookType.DINGTALK:
      return create(AppIMSetting_IMSettingSchema, {
        type: wt,
        payload: {
          case: "dingtalk",
          value: create(AppIMSetting_DingTalkSchema, init),
        },
      });
    case WebhookType.TEAMS:
      return create(AppIMSetting_IMSettingSchema, {
        type: wt,
        payload: {
          case: "teams",
          value: create(AppIMSetting_TeamsSchema, init),
        },
      });
    default:
      return create(AppIMSetting_IMSettingSchema, { type: wt });
  }
}

const UPDATE_MASKS: Record<number, string> = {
  [WebhookType.SLACK]: "value.app_im.slack",
  [WebhookType.FEISHU]: "value.app_im.feishu",
  [WebhookType.LARK]: "value.app_im.lark",
  [WebhookType.WECOM]: "value.app_im.wecom",
  [WebhookType.DINGTALK]: "value.app_im.dingtalk",
  [WebhookType.TEAMS]: "value.app_im_setting_value.teams",
};

function webhookTypeFromKey(key: string): WebhookType {
  for (const [num, k] of Object.entries(IM_TYPE_KEY)) {
    if (k === key) return Number(num) as WebhookType;
  }
  return WebhookType.WEBHOOK_TYPE_UNSPECIFIED;
}

function getStoredIMSetting() {
  const setting = settingStore.getSettingByName(Setting_SettingName.APP_IM);
  if (setting?.value?.value?.case !== "appIm") {
    return create(AppIMSettingSchema, { settings: [] });
  }
  return setting.value.value.value;
}

// Mask sensitive values for display
function maskValues(payload: Record<string, unknown>): Record<string, string> {
  const result: Record<string, string> = {};
  for (const [k, v] of Object.entries(payload)) {
    if (k === "$typeName" || k === "$unknown") continue;
    result[k] = typeof v === "string" && v !== "" ? "*********" : "";
  }
  return result;
}

const imPendingSaveType = ref<string | null>(null);

// Local IM state for tracking unsaved additions
const imLocalSettings = ref<AppIMSetting_IMSetting[]>([]);
const imInitialized = ref(false);

function syncIMLocalState() {
  const stored = getStoredIMSetting();
  imLocalSettings.value = stored.settings.map((s) =>
    clone(AppIMSetting_IMSettingSchema, s)
  );
}

function buildIMProps() {
  const stored = getStoredIMSetting();
  const configuredTypes = new Set(stored.settings.map((s) => s.type));

  const settings = imLocalSettings.value.map((item) => {
    const typeKey = IM_TYPE_KEY[item.type] ?? "";
    const fields = IM_FIELDS[item.type] ?? [];
    const payloadObj = (item.payload?.value ?? {}) as Record<string, unknown>;
    return {
      type: typeKey,
      typeLabel: IM_LABELS[item.type] ?? typeKey,
      isConfigured: configuredTypes.has(item.type),
      fields,
      values: configuredTypes.has(item.type)
        ? maskValues(payloadObj)
        : Object.fromEntries(
            fields.map((f) => [f.key, String(payloadObj[f.key] ?? "")])
          ),
    };
  });

  const existingTypes = new Set(imLocalSettings.value.map((s) => s.type));
  const availableTypes = IM_TYPES.filter(
    (wt) => !configuredTypes.has(wt) && !existingTypes.has(wt)
  ).map((wt) => ({
    type: IM_TYPE_KEY[wt],
    label: IM_LABELS[wt],
  }));

  return {
    settings,
    availableTypes,
    allowEdit: props.allowEdit,
    pendingSaveType: imPendingSaveType.value,
    onAdd: (typeKey: string) => {
      const wt = webhookTypeFromKey(typeKey);
      imLocalSettings.value = [...imLocalSettings.value, createIMSetting(wt)];
    },
    onSave: async (
      index: number,
      typeKey: string,
      values: Record<string, string>
    ) => {
      const wt = webhookTypeFromKey(typeKey);
      imPendingSaveType.value = typeKey;
      try {
        const reconstructed = createIMSetting(wt, values);
        const current = clone(AppIMSettingSchema, getStoredIMSetting());
        const existingIdx = current.settings.findIndex((s) => s.type === wt);
        if (existingIdx >= 0) {
          current.settings[existingIdx] = reconstructed;
        } else {
          current.settings.push(reconstructed);
        }
        await settingStore.upsertSetting({
          name: Setting_SettingName.APP_IM,
          value: create(SettingValueSchema, {
            value: { case: "appIm", value: current },
          }),
          updateMask: create(FieldMaskSchema, {
            paths: [UPDATE_MASKS[wt]],
          }),
        });
        syncIMLocalState();
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.updated"),
        });
      } finally {
        imPendingSaveType.value = null;
      }
    },
    onDelete: async (index: number, typeKey: string) => {
      const wt = webhookTypeFromKey(typeKey);
      const stored = getStoredIMSetting();
      const wasConfigured = stored.settings.some((s) => s.type === wt);
      if (wasConfigured) {
        await settingStore.upsertSetting({
          name: Setting_SettingName.APP_IM,
          value: create(SettingValueSchema, {
            value: {
              case: "appIm",
              value: create(AppIMSettingSchema, {
                settings: stored.settings.filter((s) => s.type !== wt),
              }),
            },
          }),
        });
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("common.deleted"),
        });
      }
      syncIMLocalState();
    },
    onChangeType: (index: number, newTypeKey: string) => {
      const wt = webhookTypeFromKey(newTypeKey);
      const next = [...imLocalSettings.value];
      next[index] = createIMSetting(wt);
      imLocalSettings.value = next;
    },
  };
}

const propsBuilders: Record<string, () => Record<string, unknown>> = {
  SubscriptionPage: buildSubscriptionProps,
  MCPPage: buildMCPProps,
  IMPage: buildIMProps,
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
  if (props.page === "IMPage") {
    return [
      settingStore.getSettingByName(Setting_SettingName.APP_IM),
      imLocalSettings.value,
      imPendingSaveType.value,
      props.allowEdit,
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

onMounted(async () => {
  if (props.page === "IMPage") {
    await settingStore.getOrFetchSettingByName(Setting_SettingName.APP_IM);
    syncIMLocalState();
    imInitialized.value = true;
  }
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
