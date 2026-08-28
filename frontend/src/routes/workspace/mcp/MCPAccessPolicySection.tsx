import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { createContextValues } from "@connectrpc/connect";
import { Rows3 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  settingServiceClientConnect,
  workspaceServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { MCPModeContentsSheet } from "@/components/mcp/MCPModeContentsSheet";
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
import { settingNamePrefix } from "@/lib/resourceName";
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
import type { MCPInfo } from "@/types/proto-es/v1/workspace_service_pb";

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

  const settingsByName = useAppStore((s) => s.settingsByName);
  // Read from the licence, not from GetMCPInfo. That request refuses outright
  // under an invalid or unserved ceiling (BOT-106), which is exactly when an
  // admin is on this page — and hiding this line there lets them arm a toggle
  // that does nothing while believing they tightened masking.
  const dataMaskingAvailable = useAppStore((s) =>
    s.hasFeature(PlanFeature.FEATURE_DATA_MASKING)
  );
  const mcpSetting = useMemo(() => {
    const setting = useAppStore
      .getState()
      .getSettingByName(Setting_SettingName.MCP);
    if (setting?.value?.value?.case === "mcp") {
      return setting.value.value.value;
    }
    return undefined;
  }, [settingsByName]);

  const [info, setInfo] = useState<MCPInfo | undefined>(undefined);
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [pick, setPick] = useState<MCPMode | undefined>(undefined);
  const [ignoreMasking, setIgnoreMasking] = useState(false);
  const [contentsFor, setContentsFor] = useState<MCPMode | undefined>(
    undefined
  );

  // GetMCPInfo resolves what a mode contains from the live descriptors, so it
  // is re-read after a save rather than patched: the ceiling in force is part
  // of the same answer.
  //
  // Silent: a ceiling this build cannot read refuses this call, and the banner
  // below already says so in the words an admin can act on. The interceptor's
  // toast would put the same fact on screen twice, once as a status code.
  // Both reads carry a generation, and the effect retires them on unmount. A
  // response has no way of knowing it was overtaken, and the store it writes is
  // shared: a read left flying by a visit the admin navigated away from would
  // otherwise land later and put that visit's row back, reverting a save made
  // since. getMCPInfo is also re-issued after every save, so its two responses
  // can land out of order on one mount.
  const infoGeneration = useRef(0);
  const settingGeneration = useRef(0);

  const loadInfo = useCallback(() => {
    const generation = ++infoGeneration.current;
    workspaceServiceClientConnect
      .getMCPInfo(
        {},
        { contextValues: createContextValues().set(silentContextKey, true) }
      )
      .then((next) => {
        if (generation === infoGeneration.current) {
          setInfo(next);
        }
      })
      .catch(() => {
        if (generation === infoGeneration.current) {
          setInfo(undefined);
        }
      });
  }, []);

  // Read the setting past the store's cache. The server reads this row uncached
  // for a reason — a hand edit or a newer replica changes it out of band — and
  // getOrFetchSettingByName returns a cached snapshot without revalidating, so
  // a second visit in one session would compute the form's dirty state against
  // a value that is no longer stored. On an invalid row that makes the
  // one-save repair unreachable without a reload.
  const [readFailed, setReadFailed] = useState(false);
  // Whether this mount's own read has answered. Until it has, a value left in
  // the store by an earlier visit is a guess, not the ceiling — so the page
  // waits rather than offering it for editing.
  const [readSettled, setReadSettled] = useState(false);
  useEffect(() => {
    setReadFailed(false);
    setReadSettled(false);
    const generation = ++settingGeneration.current;
    const settle = (failed: boolean) => {
      if (generation !== settingGeneration.current) {
        return;
      }
      setReadFailed(failed);
      setReadSettled(true);
    };
    settingServiceClientConnect
      .getSetting(
        {
          name: `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.MCP]}`,
        },
        {
          contextValues: createContextValues().set(silentContextKey, true),
          // The card waits for this read, so an unbounded one is a page that
          // never loads. The transport declares no default timeout.
          timeoutMs: 30_000,
        }
      )
      .then((setting) => {
        if (generation !== settingGeneration.current) {
          return;
        }
        useAppStore.getState().setSettingByName(setting);
        settle(false);
      })
      // A row this build cannot unmarshal fails this read. The store may still
      // hold a value from an earlier visit, and rendering that would report a
      // ceiling nobody is enforcing — the failure outranks it below.
      .catch(() => settle(true));
    loadInfo();
    // Retire both reads when this instance goes away. The generations are this
    // mount's; the setting store they write is the application's.
    return () => {
      settingGeneration.current++;
      infoGeneration.current++;
    };
  }, [loadInfo]);

  const storedCapability = mcpSetting?.capability;
  const storedMode =
    storedCapability !== undefined && isMCPMode(storedCapability)
      ? storedCapability
      : undefined;
  const unreadable =
    storedCapability === MCPSetting_Capability.CAPABILITY_UNSPECIFIED;
  const storedIgnoreMasking = mcpSetting?.ignoreMaskingExemptions ?? false;

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
      loadInfo();
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("settings.mcp.policy.saved", { mode: modeLabel(pick) }),
      });
    } finally {
      setSaving(false);
    }
  };

  // The mode the drawer is showing, frozen while it closes. The Sheet unmounts
  // after its close animation, so reading contentsFor directly would swap the
  // contents for another mode's for those ~200ms.
  const openContentsRef = useRef<MCPMode>(MCPSetting_Capability.READ_ONLY);
  if (contentsFor !== undefined) {
    openContentsRef.current = contentsFor;
  }
  const openContents = openContentsRef.current;

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
              className="grid grid-cols-1 gap-4 md:grid-cols-3"
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
                    // this the radio's name absorbs the description, the
                    // contents link and the "Best for" line.
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
                    {info && capability !== MCPSetting_Capability.DISABLED && (
                      <Button
                        appearance="link"
                        size="sm"
                        className="self-start px-0"
                        onClick={(e) => {
                          // This sits inside the card's own label, so a click
                          // here would otherwise pick the mode too.
                          e.preventDefault();
                          e.stopPropagation();
                          setContentsFor(capability);
                        }}
                      >
                        {t("settings.mcp.policy.mode.contents", {
                          mode: modeLabel(capability),
                        })}
                      </Button>
                    )}
                    {/* Pinned to the bottom so the three "Best for" lines sit
                        on one row, however long each description runs. */}
                    <p className="textinfolabel mt-auto pt-2">
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
                className="mt-0.5"
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
            <div className="flex flex-wrap items-center justify-between gap-2">
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

      {/* No info, no drawer. The trigger is already gated on it, and rendering
          without it would describe a mode that serves nothing rather than
          admitting the contents are unknown. */}
      {info && (
        <MCPModeContentsSheet
          open={contentsFor !== undefined}
          capability={openContents}
          info={info}
          modeLabel={modeLabel(openContents)}
          // While editing, the drawer previews the policy about to be saved,
          // not the one stored — the admin opens it from a card they are
          // choosing.
          ignoreMaskingExemptions={
            editing ? ignoreMasking : storedIgnoreMasking
          }
          onClose={() => setContentsFor(undefined)}
        />
      )}
    </div>
  );
}
