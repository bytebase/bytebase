import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { MCPModeBadge } from "@/components/mcp/MCPModeBadge";
import type { MCPMode } from "@/components/mcp/mcpPolicy";
import {
  isMCPMode,
  isServingMode,
  MCP_CAPABILITY_CHOICES,
  MCP_MODE_PRESENTATION,
  mcpModeKey,
} from "@/components/mcp/mcpPolicy";
import { PermissionGuard } from "@/components/PermissionGuard";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { useLocalStorageBoolean } from "@/hooks/useLocalStorageBoolean";
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
import {
  STORAGE_KEY_MCP_LADDER_DETAILS,
  STORAGE_KEY_MCP_LADDER_OPEN,
} from "@/utils/storage-keys";
import { MCPCapabilityLadder } from "./MCPCapabilityLadder";

export function MCPAccessPolicySection() {
  const { t } = useTranslation();

  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [pick, setPick] = useState<MCPMode | undefined>(undefined);
  const [ignoreMasking, setIgnoreMasking] = useState(false);
  const [readFailed, setReadFailed] = useState(false);
  const [readSettled, setReadSettled] = useState(false);
  // A habit of the person, not a fact about the workspace: an admin who opened
  // the list once wants it open the next time they come to compare.
  const [ladderOpen, setLadderOpen] = useLocalStorageBoolean(
    STORAGE_KEY_MCP_LADDER_OPEN,
    false
  );
  const [ladderDetails, setLadderDetails] = useLocalStorageBoolean(
    STORAGE_KEY_MCP_LADDER_DETAILS,
    false
  );
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

  // The masking toggle governs what an MCP session may unmask, and only a
  // serving mode admits one: mcpIgnoresMaskingExemptions answers on the
  // delegated grant an MCP request carries, so the stored flag is never read
  // under Disabled — nor under a ceiling nobody has picked yet. The control is
  // withheld there because it governs nothing there.
  //
  // The draft it holds is kept, and saved, whatever the pick. Withholding a
  // control is not a reason to discard what the admin set with it: clicking
  // through the modes to read their descriptions must not silently undo an
  // unrelated edit, and under Disabled the stored flag is inert rather than
  // wrong, so writing it costs nothing now and honors the choice when MCP is
  // turned back on.
  const maskingApplies = isServingMode(pick);
  const maskingChanged = ignoreMasking !== storedIgnoreMasking;
  const isDirty = editing && (pick !== storedMode || maskingChanged);
  useUnsavedChangesGuard(isDirty);
  // A row nobody can read is repaired by naming a capability. Saving anything
  // else would erase it, and the server refuses that write.
  const canSave = isDirty && pick !== undefined;

  const modeLabel = (capability: MCPMode): string =>
    t(mcpModeKey(capability, "title"));

  const save = async () => {
    if (pick === undefined) {
      return;
    }
    const paths: string[] = [];
    if (pick !== storedMode) {
      paths.push("value.mcp.capability");
    }
    if (maskingChanged) {
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

  // Disabled has no list, so it says its one sentence instead. Every other mode
  // discloses the same ladder in view and in edit; picking a mode while editing
  // renders exactly what the view will show once it is saved.
  const disclosure = (mode: MCPMode) => {
    if (!isServingMode(mode)) {
      return editing ? (
        <p className="rounded-sm bg-error/5 px-3 py-2 text-sm text-error">
          {t("settings.mcp.ladder.disabled")}
        </p>
      ) : (
        <p className="textinfolabel">
          {t("settings.mcp.policy.mode.disabled.description")}
        </p>
      );
    }
    return (
      <MCPCapabilityLadder
        mode={mode}
        expanded={ladderOpen}
        details={ladderDetails}
        onExpandedChange={setLadderOpen}
        onDetailsChange={setLadderDetails}
      />
    );
  };

  // The footer names the change while the form is dirty, so an admin reads the
  // transition they are about to apply rather than a general rule. A repair of
  // an unreadable row has no "from" to name, so it keeps the plain sentence.
  const footerSentence = () =>
    pick !== undefined && storedMode !== undefined && pick !== storedMode
      ? t("settings.mcp.policy.tightening-change", {
          from: modeLabel(storedMode),
          to: modeLabel(pick),
        })
      : t("settings.mcp.policy.tightening");

  // A masking edit survives a pick that hides its control, and is written with
  // the rest. Nothing else on the card would say so, so the footer does.
  const maskingPending = maskingChanged && !maskingApplies;

  // Whether the stored flag is doing anything, and if not, why not.
  const maskingInEffect = isServingMode(storedMode) && dataMaskingAvailable;
  // Resolved here rather than threaded as a key, so each stays a literal
  // translation call the unused-key checker can trace.
  const maskingBadgeText = !isServingMode(storedMode)
    ? t("settings.mcp.policy.masking.badge-disabled")
    : dataMaskingAvailable
      ? t("settings.mcp.policy.masking.badge")
      : t("settings.mcp.policy.masking.badge-unlicensed");

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
        {!editing && (
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
                <MCPModeBadge
                  mode={storedMode}
                  describedAs={t("settings.mcp.policy.current", {
                    mode: modeLabel(storedMode),
                  })}
                />
                {/* Shown wherever the flag is stored, including the states
                    where it does nothing — hiding it would leave a set flag
                    with nowhere to see it, and a masking edit survives a pick
                    that hides its control, so Disabled is a state it reaches.
                    Each inert case says why it is inert: the reasons live in
                    the editor and in the Disabled sentence, and the editor is a
                    different branch behind bb.settings.set that a reader of
                    this page may never reach. */}
                {storedIgnoreMasking && (
                  <Badge variant={maskingInEffect ? "secondary" : "default"}>
                    {maskingBadgeText}
                  </Badge>
                )}
              </div>
            )}
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
          </div>
        )}

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
              className="grid grid-cols-1 items-stretch gap-2 sm:grid-cols-3"
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
                const { icon: Icon } = MCP_MODE_PRESENTATION[capability];
                const picked = pick === capability;
                return (
                  <RadioGroupItem
                    key={capability}
                    value={String(capability)}
                    // The item wraps the whole card in a label, so without this
                    // the radio's name absorbs the caption too.
                    aria-label={modeLabel(capability)}
                    className={cn(
                      "h-full rounded-sm border px-3 py-2",
                      "has-[:focus-visible]:outline-hidden has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-accent has-[:focus-visible]:ring-offset-2",
                      picked
                        ? "border-accent bg-accent/5 ring-1 ring-accent"
                        : "border-control-border hover:border-accent/50 hover:bg-control-bg"
                    )}
                    contentClassName="flex min-w-0 items-center gap-x-2"
                    // Hidden rather than placed: the card is the control, and
                    // the label carries the focus ring for it.
                    radioClassName="sr-only"
                  >
                    <Icon
                      className={cn(
                        "size-5 shrink-0",
                        picked ? "text-accent" : "text-control-light"
                      )}
                    />
                    <span className="flex min-w-0 flex-col">
                      <span className="text-sm font-medium text-main">
                        {modeLabel(capability)}
                      </span>
                      <span className="text-xs text-control-light">
                        {t(mcpModeKey(capability, "caption"))}
                      </span>
                    </span>
                  </RadioGroupItem>
                );
              })}
            </RadioGroup>

            {pick === undefined ? (
              <p className="text-sm text-warning">
                {t("settings.mcp.policy.unreadable.pick")}
              </p>
            ) : (
              <>
                <p className="textinfolabel">
                  {t(mcpModeKey(pick, "best-for"))}
                </p>
                {disclosure(pick)}
              </>
            )}

            {maskingApplies && (
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
                  {!dataMaskingAvailable && (
                    <div className="text-sm text-warning">
                      {t("settings.mcp.policy.masking.unavailable")}
                    </div>
                  )}
                </div>
              </div>
            )}

            <Separator />
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div className="flex flex-col gap-1">
                <p className="textinfolabel">{footerSentence()}</p>
                {maskingPending && (
                  <p className="textinfolabel">
                    {t("settings.mcp.policy.masking-pending")}
                  </p>
                )}
              </div>
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
          storedMode !== undefined && disclosure(storedMode)
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
