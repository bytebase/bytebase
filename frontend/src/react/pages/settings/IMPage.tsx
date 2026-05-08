import { clone, create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Plus, Trash2 } from "lucide-react";
import { type ChangeEvent, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
// Static image imports — Vite cannot resolve dynamic src in <img>
import dingtalkIcon from "@/assets/im/dingtalk.png";
import feishuIcon from "@/assets/im/feishu.webp";
import slackIcon from "@/assets/im/slack.png";
import teamsIcon from "@/assets/im/teams.svg";
import wecomIcon from "@/assets/im/wecom.png";
import { PermissionGuard } from "@/react/components/PermissionGuard";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { useVueState } from "@/react/hooks/useVueState";
import { pushNotification, useSettingV1Store } from "@/store";
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
import { hasWorkspacePermissionV2 } from "@/utils";

// --- Constants ---

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

const IM_ICON_MAP: Record<string, string> = {
  SLACK: slackIcon,
  FEISHU: feishuIcon,
  LARK: feishuIcon,
  WECOM: wecomIcon,
  DINGTALK: dingtalkIcon,
  TEAMS: teamsIcon,
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

const UPDATE_MASKS: Record<number, string> = {
  [WebhookType.SLACK]: "value.app_im.slack",
  [WebhookType.FEISHU]: "value.app_im.feishu",
  [WebhookType.LARK]: "value.app_im.lark",
  [WebhookType.WECOM]: "value.app_im.wecom",
  [WebhookType.DINGTALK]: "value.app_im.dingtalk",
  [WebhookType.TEAMS]: "value.app_im_setting_value.teams",
};

// --- Protobuf helpers ---

function webhookTypeFromKey(key: string): WebhookType {
  for (const [num, k] of Object.entries(IM_TYPE_KEY)) {
    if (k === key) return Number(num) as WebhookType;
  }
  return WebhookType.WEBHOOK_TYPE_UNSPECIFIED;
}

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

function maskValues(payload: Record<string, unknown>): Record<string, string> {
  const result: Record<string, string> = {};
  for (const [k, v] of Object.entries(payload)) {
    if (k === "$typeName" || k === "$unknown") continue;
    result[k] = typeof v === "string" && v !== "" ? "*********" : "";
  }
  return result;
}

function getStoredIMSetting(
  settingStore: ReturnType<typeof useSettingV1Store>
) {
  const setting = settingStore.getSettingByName(Setting_SettingName.APP_IM);
  if (setting?.value?.value?.case !== "appIm") {
    return create(AppIMSettingSchema, { settings: [] });
  }
  return setting.value.value.value;
}

// --- Derived view model ---

interface IMSettingView {
  type: string;
  typeLabel: string;
  isConfigured: boolean;
  fields: { key: string; label: string }[];
  values: Record<string, string>;
}

function buildSettingsView(
  localSettings: AppIMSetting_IMSetting[],
  configuredTypes: Set<number>
): IMSettingView[] {
  return localSettings.map((item) => {
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
}

// --- Component ---

export function IMPage() {
  const { t } = useTranslation();
  const settingStore = useSettingV1Store();
  const allowEdit = hasWorkspacePermissionV2("bb.settings.set");

  // Subscribe to store changes
  const storedSetting = useVueState(() => getStoredIMSetting(settingStore));

  // Local state
  const [localSettings, setLocalSettings] = useState<AppIMSetting_IMSetting[]>(
    []
  );
  const [localValues, setLocalValues] = useState<Record<string, string>[]>([]);
  const [pendingSaveType, setPendingSaveType] = useState<string | null>(null);
  const [initialized, setInitialized] = useState(false);

  // Fetch on mount
  useEffect(() => {
    settingStore
      .getOrFetchSettingByName(Setting_SettingName.APP_IM)
      .then(() => setInitialized(true));
  }, [settingStore]);

  // Sync local state from store
  const syncFromStore = useCallback(() => {
    const stored = getStoredIMSetting(settingStore);
    const cloned = stored.settings.map((s) =>
      clone(AppIMSetting_IMSettingSchema, s)
    );
    setLocalSettings(cloned);
  }, [settingStore]);

  useEffect(() => {
    if (initialized) syncFromStore();
  }, [initialized, storedSetting, syncFromStore]);

  // Derive view model
  const configuredTypes = new Set(storedSetting.settings.map((s) => s.type));
  const settingsView = buildSettingsView(localSettings, configuredTypes);

  // Sync form values when settings view changes
  useEffect(() => {
    setLocalValues(settingsView.map((s) => ({ ...s.values })));
  }, [settingsView.length, initialized]);

  const existingTypes = new Set(localSettings.map((s) => s.type));
  const availableTypes = IM_TYPES.filter(
    (wt) => !configuredTypes.has(wt) && !existingTypes.has(wt)
  ).map((wt) => ({ type: IM_TYPE_KEY[wt], label: IM_LABELS[wt] }));

  // --- Handlers ---

  const handleAdd = (typeKey: string) => {
    const wt = webhookTypeFromKey(typeKey);
    setLocalSettings((prev) => [...prev, createIMSetting(wt)]);
  };

  const handleChangeType = (index: number, newTypeKey: string) => {
    const wt = webhookTypeFromKey(newTypeKey);
    setLocalSettings((prev) => {
      const next = [...prev];
      next[index] = createIMSetting(wt);
      return next;
    });
  };

  const updateField = (index: number, key: string, value: string) => {
    setLocalValues((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], [key]: value };
      return next;
    });
  };

  const isDirty = (index: number) => {
    const local = localValues[index];
    const original = settingsView[index]?.values;
    if (!local || !original) return false;
    return Object.keys(original).some((k) => local[k] !== original[k]);
  };

  const handleDiscard = (index: number) => {
    const original = settingsView[index];
    if (!original) return;
    // If not configured (new item), remove it
    if (!original.isConfigured) {
      setLocalSettings((prev) => prev.filter((_, i) => i !== index));
    } else {
      setLocalValues((prev) => {
        const next = [...prev];
        next[index] = { ...original.values };
        return next;
      });
    }
  };

  const handleSave = async (index: number, typeKey: string) => {
    const wt = webhookTypeFromKey(typeKey);
    const values = localValues[index] ?? {};
    setPendingSaveType(typeKey);
    try {
      const reconstructed = createIMSetting(wt, values);
      const current = clone(
        AppIMSettingSchema,
        getStoredIMSetting(settingStore)
      );
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
        updateMask: create(FieldMaskSchema, { paths: [UPDATE_MASKS[wt]] }),
      });
      syncFromStore();
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } finally {
      setPendingSaveType(null);
    }
  };

  const handleDelete = async (index: number, typeKey: string) => {
    if (!window.confirm(t("bbkit.confirm-button.sure-to-delete"))) return;
    const wt = webhookTypeFromKey(typeKey);
    const stored = getStoredIMSetting(settingStore);
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
    syncFromStore();
  };

  // --- Render ---

  if (!initialized) return null;

  if (settingsView.length === 0) {
    return (
      <div className="w-full px-4 flex flex-col gap-y-4 py-4">
        <Description />
        <div className="py-12 border rounded-sm flex flex-col items-center justify-center gap-y-4 text-control-light">
          <span>{t("common.no-data")}</span>
          {availableTypes.length > 0 && (
            <PermissionGuard permissions={["bb.settings.set"]}>
              <Button
                disabled={!allowEdit}
                onClick={() => handleAdd(availableTypes[0].type)}
              >
                <Plus className="h-4 w-4 mr-1" />
                {t("common.create")}
              </Button>
            </PermissionGuard>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="w-full px-4 flex flex-col gap-y-4 py-4">
      <Description />

      {settingsView.map((item, i) => (
        <div key={item.type} className="border rounded-sm p-4">
          {item.isConfigured ? (
            <IMTypeLabel type={item.type} label={item.typeLabel} />
          ) : (
            <select
              value={item.type}
              className="flex h-9 w-full max-w-xs rounded-xs border border-control-border bg-transparent px-3 py-1 text-sm cursor-pointer"
              onChange={(e) => handleChangeType(i, e.target.value)}
            >
              <option value={item.type}>{item.typeLabel}</option>
              {availableTypes
                .filter((opt) => opt.type !== item.type)
                .map((opt) => (
                  <option key={opt.type} value={opt.type}>
                    {opt.label}
                  </option>
                ))}
            </select>
          )}

          <div className="mt-4 flex flex-col gap-y-4">
            {item.fields.map((field) => (
              <div key={field.key}>
                <label className="textlabel">{field.label}</label>
                <Input
                  className="mt-2"
                  disabled={!allowEdit}
                  placeholder={t("common.sensitive-placeholder")}
                  value={localValues[i]?.[field.key] ?? ""}
                  onChange={(e: ChangeEvent<HTMLInputElement>) =>
                    updateField(i, field.key, e.target.value)
                  }
                />
              </div>
            ))}
          </div>

          <div className="flex items-center justify-between mt-4 gap-x-2">
            <div>
              {item.isConfigured && allowEdit && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-error hover:text-error hover:bg-error/10"
                  onClick={() => handleDelete(i, item.type)}
                >
                  <Trash2 className="w-4 h-4" />
                </Button>
              )}
            </div>
            {isDirty(i) && (
              <div className="flex items-center gap-x-2">
                <Button
                  variant="outline"
                  disabled={!!pendingSaveType}
                  onClick={() => handleDiscard(i)}
                >
                  {t("common.discard-changes")}
                </Button>
                <Button
                  disabled={!!pendingSaveType || !allowEdit}
                  onClick={() => handleSave(i, item.type)}
                >
                  {t("common.save")}
                </Button>
              </div>
            )}
          </div>
        </div>
      ))}

      {availableTypes.length > 0 && (
        <div className="flex justify-end">
          <PermissionGuard permissions={["bb.settings.set"]}>
            <Button
              variant="outline"
              disabled={!allowEdit}
              onClick={() => handleAdd(availableTypes[0].type)}
            >
              <Plus className="h-4 w-4 mr-1" />
              {t("common.create")}
            </Button>
          </PermissionGuard>
        </div>
      )}
    </div>
  );
}

function Description() {
  const { t } = useTranslation();
  return (
    <div className="textinfolabel">
      {t("settings.im-integration.description")}{" "}
      <a
        href="https://docs.bytebase.com/change-database/webhook?source=console"
        target="_blank"
        rel="noopener noreferrer"
        className="text-accent hover:underline"
      >
        {t("common.learn-more")} &gt;
      </a>
    </div>
  );
}

function IMTypeLabel({ type, label }: { type: string; label: string }) {
  const icon = IM_ICON_MAP[type];
  return (
    <div className="flex items-center gap-x-2">
      {icon && <img src={icon} alt={label} className="h-5 w-5" />}
      <span className="text-sm font-medium">{label}</span>
    </div>
  );
}
