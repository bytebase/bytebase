import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Rows3 } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import type { MCPMode } from "@/components/mcp/mcpPolicy";
import { isMCPMode, MCP_CAPABILITY_CHOICES } from "@/components/mcp/mcpPolicy";
import { PermissionGuard } from "@/components/PermissionGuard";
import { Alert } from "@/components/ui/alert";
import type { BadgeProps } from "@/components/ui/badge";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import { cn } from "@/lib/utils";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import {
  MCPSetting_Capability,
  MCPSettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

// One row per ceiling an admin can pick: the locale-key stem, the glyph tone,
// and the chip variant. The tone and the variant always agree, and carry from
// the card to the in-force chip, so they belong on one row rather than in
// parallel tables that can drift apart.
const MODES: Record<
  MCPMode,
  { key: string; tone: string; badge: BadgeProps["variant"] }
> = {
  [MCPSetting_Capability.DISABLED]: {
    key: "disabled",
    tone: "text-error",
    badge: "destructive",
  },
  [MCPSetting_Capability.READ_ONLY]: {
    key: "read-only",
    tone: "text-success",
    badge: "success",
  },
  [MCPSetting_Capability.READ_WRITE]: {
    key: "read-write",
    tone: "text-warning",
    badge: "warning",
  },
};

export function MCPAccessPolicySection() {
  const { t } = useTranslation();

  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [pick, setPick] = useState<MCPMode | undefined>(undefined);
  const [ignoreMasking, setIgnoreMasking] = useState(false);
  const [readFailed, setReadFailed] = useState(false);
  const [readSettled, setReadSettled] = useState(false);
  const serverInfo = useAppStore((state) => state.serverInfo);
  const loadServerInfo = useAppStore((state) => state.loadServerInfo);
  const refreshServerInfo = useAppStore((state) => state.refreshServerInfo);
  const dataMaskingAvailable = useAppStore((state) =>
    state.hasFeature(PlanFeature.FEATURE_DATA_MASKING)
  );

  useEffect(() => {
    void loadServerInfo().then((info) => {
      setReadFailed(!info?.mcpSetting);
      setReadSettled(true);
    });
  }, [loadServerInfo]);

  const storedCapability = serverInfo?.mcpSetting?.capability;
  const storedMode =
    storedCapability !== undefined && isMCPMode(storedCapability)
      ? storedCapability
      : undefined;
  const unreadable =
    storedCapability === MCPSetting_Capability.CAPABILITY_UNSPECIFIED;
  const storedIgnoreMasking =
    serverInfo?.mcpSetting?.ignoreMaskingExemptions ?? false;

  // The form is seeded when editing opens, not on every store change: the
  // stored value only moves under an open form when someone else saved, and
  // replacing an admin's unsaved pick is worse than showing it stale.
  const startEditing = () => {
    setPick(storedMode);
    setIgnoreMasking(storedIgnoreMasking);
    setEditing(true);
  };

  const isDirty =
    editing && (pick !== storedMode || ignoreMasking !== storedIgnoreMasking);
  // The section this replaced was registered in GeneralPage's guarded refs, so
  // moving it to its own route would otherwise drop the confirm an admin gets
  // when navigating away from an unsaved ceiling.
  useUnsavedChangesGuard(isDirty);
  // A row nobody can read is repaired by naming a capability. Saving anything
  // else would erase it, and the server refuses that write.
  const canSave = isDirty && pick !== undefined;

  const modeLabel = (capability: MCPMode): string =>
    t(`settings.mcp.policy.mode.${MODES[capability].key}.title`);

  const save = async () => {
    if (pick === undefined) {
      return;
    }
    const paths: string[] = [];
    if (pick !== storedMode) {
      paths.push("value.mcp.capability");
    }
    if (ignoreMasking !== storedIgnoreMasking) {
      paths.push("value.mcp.ignore_masking_exemptions");
    }
    setSaving(true);
    try {
      await useAppStore.getState().upsertSetting({
        name: Setting_SettingName.MCP,
        value: create(SettingValueSchema, {
          value: {
            case: "mcp",
            value: create(MCPSettingSchema, {
              capability: pick,
              ignoreMaskingExemptions: ignoreMasking,
            }),
          },
        }),
        updateMask: create(FieldMaskSchema, { paths }),
      });
      setEditing(false);
      await refreshServerInfo();
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("settings.mcp.policy.saved", { mode: modeLabel(pick) }),
      });
    } finally {
      setSaving(false);
    }
  };

  // Three states share this slot and only the last renders a policy. Early
  // returns rather than a ternary chain, so each state is named where it is
  // decided and the card reads as the ordinary case it is.
  const policyBody = () => {
    if (readFailed) {
      return (
        <Alert
          variant="error"
          title={t("settings.mcp.policy.read-failed.title")}
          description={t("settings.mcp.policy.read-failed.description")}
        />
      );
    }
    if (!readSettled || storedCapability === undefined) {
      return (
        <p className="textinfolabel">{t("settings.mcp.policy.loading")}</p>
      );
    }
    return (
      <div className="rounded-sm border border-control-border p-4 flex flex-col gap-y-4">
        <div className="flex items-start justify-between gap-x-2">
          {storedMode === undefined ? (
            <span className="text-sm font-medium text-warning">
              {t(
                unreadable
                  ? "settings.mcp.policy.unreadable.title"
                  : "settings.mcp.policy.unserved.title"
              )}
            </span>
          ) : (
            <div className="flex flex-wrap items-center gap-2">
              <Rows3
                className={cn("size-4 shrink-0", MODES[storedMode].tone)}
              />
              <span className="text-sm text-control-light">
                {t("settings.mcp.policy.in-force")}
              </span>
              <Badge variant={MODES[storedMode].badge}>
                {modeLabel(storedMode)}
              </Badge>
              {storedIgnoreMasking && (
                <Badge variant="secondary">
                  {t("settings.mcp.policy.masking.badge")}
                </Badge>
              )}
            </div>
          )}
          {!editing && (
            <PermissionGuard permissions={["bb.settings.set"]}>
              {({ disabled }) => (
                <Button
                  appearance="outline"
                  size="sm"
                  disabled={disabled}
                  onClick={startEditing}
                >
                  {t("settings.mcp.policy.edit")}
                </Button>
              )}
            </PermissionGuard>
          )}
        </div>

        {storedMode === undefined && (
          <Alert
            variant="warning"
            description={
              unreadable
                ? t("settings.mcp.policy.unreadable.description")
                : t("settings.mcp.policy.unserved.description", {
                    stored: String(storedCapability),
                  })
            }
          />
        )}

        {editing ? (
          <>
            {/* Locked while the save is out: the request already captured
                this draft, so a later edit would change what is on screen and
                nothing else, then vanish when the editor closes. */}
            <RadioGroup
              aria-label={t("settings.mcp.policy.title")}
              className="grid grid-cols-1 gap-4 lg:grid-cols-3"
              disabled={saving}
              value={pick === undefined ? "" : String(pick)}
              onValueChange={(value) => {
                const capability = Number(value) as MCPSetting_Capability;
                if (isMCPMode(capability)) {
                  setPick(capability);
                }
              }}
            >
              {MCP_CAPABILITY_CHOICES.map((capability) => {
                const mode = MODES[capability];
                return (
                  <RadioGroupItem
                    key={capability}
                    value={String(capability)}
                    // The item wraps the whole card in a label, so without
                    // this the radio's name absorbs the description and the
                    // "Best for" line.
                    aria-label={t(`settings.mcp.policy.mode.${mode.key}.title`)}
                    className={cn(
                      "relative h-full flex-col items-stretch rounded-sm border p-4",
                      pick === capability
                        ? "border-accent"
                        : "border-control-border"
                    )}
                    contentClassName="flex h-full flex-col gap-2"
                    radioClassName="absolute right-4 top-4"
                  >
                    <div className="flex items-center gap-x-2 pr-6">
                      <Rows3 className={cn("size-4 shrink-0", mode.tone)} />
                      <span className="textinfo font-semibold">
                        {t(`settings.mcp.policy.mode.${mode.key}.title`)}
                      </span>
                    </div>
                    <p className="textinfolabel">
                      {t(`settings.mcp.policy.mode.${mode.key}.description`)}
                    </p>
                    <p className="textinfolabel pt-2">
                      {t(`settings.mcp.policy.mode.${mode.key}.best-for`)}
                    </p>
                  </RadioGroupItem>
                );
              })}
            </RadioGroup>

            <div className="flex items-start gap-x-3">
              <Switch
                checked={ignoreMasking}
                onCheckedChange={setIgnoreMasking}
                disabled={saving}
                aria-label={t("settings.mcp.policy.masking.title")}
                className="mt-0.5 shrink-0"
              />
              <div className="flex flex-col gap-1">
                <div className="textinfo font-semibold">
                  {t("settings.mcp.policy.masking.title")}
                </div>
                <div className="textinfolabel">
                  {t("settings.mcp.policy.masking.description")}
                </div>
                <div className="textinfolabel">
                  {t("settings.mcp.policy.masking.limits")}
                </div>
                {!dataMaskingAvailable && (
                  <div className="text-sm text-warning">
                    {t("settings.mcp.policy.masking.unavailable")}
                  </div>
                )}
              </div>
            </div>

            {pick === undefined && (
              <p className="text-sm text-warning">
                {t("settings.mcp.policy.unreadable.pick")}
              </p>
            )}

            <Separator />
            <div className="flex flex-wrap items-center justify-between gap-4">
              <p className="textinfolabel">
                {t("settings.mcp.policy.tightening")}
              </p>
              <div className="flex shrink-0 gap-x-2">
                <Button
                  appearance="outline"
                  disabled={saving}
                  onClick={() => setEditing(false)}
                >
                  {t("common.cancel")}
                </Button>
                <Button disabled={!canSave || saving} onClick={save}>
                  {t("settings.mcp.policy.save")}
                </Button>
              </div>
            </div>
          </>
        ) : (
          <>
            {storedMode !== undefined && (
              <p className="textinfolabel">
                {t(
                  `settings.mcp.policy.mode.${
                    MODES[storedMode].key
                  }.description`
                )}
              </p>
            )}
            <div className="flex flex-col gap-y-1">
              <p className="textinfolabel">
                {t("settings.mcp.policy.tightening")}
              </p>
              <p className="textinfolabel">{t("settings.mcp.policy.audit")}</p>
            </div>
          </>
        )}
      </div>
    );
  };

  return (
    <div className="flex flex-col gap-y-3">
      <div className="flex flex-col gap-y-1">
        <h3 className="text-base font-medium">
          {t("settings.mcp.policy.title")}
        </h3>
        <p className="textinfolabel">{t("settings.mcp.policy.description")}</p>
      </div>

      {policyBody()}
    </div>
  );
}
