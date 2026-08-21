import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/components/PermissionGuard";
import { FormField, FormFieldGroup, FormSection } from "@/components/ui/form";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { useAppStore } from "@/stores/app";
import {
  MCPSetting_Capability,
  MCPSettingSchema,
  Setting_SettingName,
  SettingValueSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import type { SectionHandle } from "./useSettingSection";

interface MCPSectionProps {
  title: string;
  onDirtyChange: () => void;
}

export interface LocalState {
  mcpCapability: MCPSetting_Capability;
}

// Unset resolves to READ_WRITE server-side, so it renders as Read-write.
// Unknown or reserved capability numbers fail closed at /mcp, so they render as
// Disabled — the option matching their actual behavior; selecting Read-only
// or Read-write then persists a valid value. A stored capability name this
// build does not know never reaches that arm: protojson discards it, so it
// arrives as unset and renders Read-write while /mcp fails closed (BOT-100).
export const normalizeCapability = (
  capability: MCPSetting_Capability
): MCPSetting_Capability => {
  switch (capability) {
    case MCPSetting_Capability.CAPABILITY_UNSPECIFIED:
    case MCPSetting_Capability.READ_WRITE:
      return MCPSetting_Capability.READ_WRITE;
    case MCPSetting_Capability.DISABLED:
    case MCPSetting_Capability.READ_ONLY:
      return capability;
    default:
      return MCPSetting_Capability.DISABLED;
  }
};

/**
 * Re-hydration rule for the form. The store value arrives after the first
 * render, so hydrating unconditionally would replace a capability the admin
 * had already picked in between — and clear the dirty footer that is their
 * only way to save it. `hydratedFrom` is what the form was last hydrated to,
 * so "unedited" is a comparison against that rather than against the value
 * just fetched.
 */
export const hydrateWhilePristine = (
  current: LocalState,
  hydratedFrom: LocalState,
  next: LocalState
): LocalState => (isEqual(current, hydratedFrom) ? next : current);

export const MCPSection = forwardRef<SectionHandle, MCPSectionProps>(
  function MCPSection({ title, onDirtyChange }, ref) {
    const { t } = useTranslation();

    const [canEdit] = usePermissionCheck(["bb.settings.set"]);

    const settingsByName = useAppStore((s) => s.settingsByName);
    const mcpSetting = useMemo(() => {
      const setting = useAppStore
        .getState()
        .getSettingByName(Setting_SettingName.MCP);
      if (setting?.value?.value?.case === "mcp") {
        return setting.value.value.value;
      }
      return undefined;
    }, [settingsByName]);

    const getInitialState = useCallback(
      (): LocalState => ({
        mcpCapability: normalizeCapability(
          mcpSetting?.capability ?? MCPSetting_Capability.CAPABILITY_UNSPECIFIED
        ),
      }),
      [mcpSetting]
    );

    const [state, setState] = useState<LocalState>(getInitialState);

    // Re-sync state when the store value changes (e.g. after the initial
    // fetch, or after a save). Only a pristine form is re-hydrated: the fetch
    // lands after the first render, so an admin who picks a capability in
    // between would otherwise have it replaced by the fetched value — and the
    // dirty footer that is their only way to save it would clear with it.
    const prevMcpSettingRef = useRef(mcpSetting);
    const hydratedRef = useRef<LocalState>(state);
    useEffect(() => {
      if (prevMcpSettingRef.current === mcpSetting) {
        return;
      }
      prevMcpSettingRef.current = mcpSetting;
      const hydrated = getInitialState();
      setState((current) =>
        hydrateWhilePristine(current, hydratedRef.current, hydrated)
      );
      hydratedRef.current = hydrated;
    }, [mcpSetting, getInitialState]);

    // Fetch setting on mount.
    useEffect(() => {
      useAppStore
        .getState()
        .getOrFetchSettingByName(Setting_SettingName.MCP, true);
    }, []);

    const isDirty = useCallback(
      () => !isEqual(state, getInitialState()),
      [state, getInitialState]
    );

    const revert = useCallback(() => {
      setState(getInitialState());
    }, [getInitialState]);

    const update = useCallback(async () => {
      const initState = getInitialState();
      if (state.mcpCapability !== initState.mcpCapability) {
        // The server merges the named path onto the stored row, so
        // ignore_masking_exemptions survives a capability-only write.
        await useAppStore.getState().upsertSetting({
          name: Setting_SettingName.MCP,
          value: create(SettingValueSchema, {
            value: {
              case: "mcp",
              value: create(MCPSettingSchema, {
                capability: state.mcpCapability,
              }),
            },
          }),
          updateMask: create(FieldMaskSchema, {
            paths: ["value.mcp.capability"],
          }),
        });
      }
    }, [state, getInitialState]);

    useImperativeHandle(
      ref,
      () => ({
        isDirty,
        revert,
        update,
      }),
      [isDirty, revert, update]
    );

    // Notify parent when state changes.
    useEffect(() => {
      onDirtyChange();
    }, [state, onDirtyChange]);

    const options = [
      {
        capability: MCPSetting_Capability.DISABLED,
        label: t("settings.general.workspace.mcp.capability.disabled.self"),
        description: t(
          "settings.general.workspace.mcp.capability.disabled.description"
        ),
      },
      {
        capability: MCPSetting_Capability.READ_ONLY,
        label: t("settings.general.workspace.mcp.capability.read-only.self"),
        description: t(
          "settings.general.workspace.mcp.capability.read-only.description"
        ),
      },
      {
        capability: MCPSetting_Capability.READ_WRITE,
        label: t("settings.general.workspace.mcp.capability.read-write.self"),
        description: t(
          "settings.general.workspace.mcp.capability.read-write.description"
        ),
      },
    ];

    return (
      <FormSection id="mcp" title={title}>
        <PermissionGuard permissions={["bb.settings.set"]} display="block">
          <FormFieldGroup>
            <FormField
              title={t("settings.general.workspace.mcp.capability.self")}
              description={t(
                "settings.general.workspace.mcp.capability.description"
              )}
              className="gap-y-4"
            >
              <RadioGroup
                className="flex-col items-stretch gap-4"
                value={String(state.mcpCapability)}
                onValueChange={(value) =>
                  setState({ mcpCapability: Number(value) })
                }
              >
                {options.map((option) => (
                  <RadioGroupItem
                    key={option.capability}
                    value={String(option.capability)}
                    disabled={!canEdit || mcpSetting === undefined}
                    className="items-start gap-x-3"
                    contentClassName="flex flex-col gap-1"
                    radioClassName="mt-1"
                  >
                    <div className="flex flex-col gap-1">
                      <div className="textinfo font-semibold">
                        {option.label}
                      </div>
                      <div className="textinfolabel">{option.description}</div>
                    </div>
                  </RadioGroupItem>
                ))}
              </RadioGroup>
            </FormField>
          </FormFieldGroup>
        </PermissionGuard>
      </FormSection>
    );
  }
);
