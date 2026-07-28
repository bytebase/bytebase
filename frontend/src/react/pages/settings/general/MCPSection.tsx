import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { isEqual } from "lodash-es";
import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import {
  PermissionGuard,
  usePermissionCheck,
} from "@/react/components/PermissionGuard";
import {
  FormField,
  FormFieldGroup,
  FormSection,
} from "@/react/components/ui/form";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import { useAppStore } from "@/react/stores/app";
import { WorkspaceProfileSetting_MCPCapability } from "@/types/proto-es/v1/setting_service_pb";
import type { SectionHandle } from "./useSettingSection";

interface MCPSectionProps {
  title: string;
  onDirtyChange: () => void;
}

interface LocalState {
  mcpCapability: WorkspaceProfileSetting_MCPCapability;
}

// Unset resolves to READ_WRITE server-side, so it renders as Read-write.
// METADATA_ONLY is not offered in the picker (it has no enforcement yet) and
// renders as Read-only, the option with the same current behavior: the server
// refuses MCP connections for both until fine-grained enforcement ships.
const normalizeCapability = (
  capability: WorkspaceProfileSetting_MCPCapability
): WorkspaceProfileSetting_MCPCapability => {
  switch (capability) {
    case WorkspaceProfileSetting_MCPCapability.MCP_DISABLED:
      return capability;
    case WorkspaceProfileSetting_MCPCapability.MCP_METADATA_ONLY:
    case WorkspaceProfileSetting_MCPCapability.MCP_READ_ONLY:
      return WorkspaceProfileSetting_MCPCapability.MCP_READ_ONLY;
    default:
      return WorkspaceProfileSetting_MCPCapability.MCP_READ_WRITE;
  }
};

export const MCPSection = forwardRef<SectionHandle, MCPSectionProps>(
  function MCPSection({ title, onDirtyChange }, ref) {
    const { t } = useTranslation();

    const [canEdit] = usePermissionCheck(["bb.settings.setWorkspaceProfile"]);

    const getInitialState = useCallback(
      (): LocalState => ({
        mcpCapability: normalizeCapability(
          useAppStore.getState().getWorkspaceProfile().mcpCapability
        ),
      }),
      []
    );

    const [state, setState] = useState<LocalState>(getInitialState);

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
        await useAppStore.getState().updateWorkspaceProfile({
          payload: {
            mcpCapability: state.mcpCapability,
          },
          updateMask: create(FieldMaskSchema, {
            paths: ["value.workspace_profile.mcp_capability"],
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
        capability: WorkspaceProfileSetting_MCPCapability.MCP_DISABLED,
        label: t("settings.general.workspace.mcp.capability.disabled.self"),
        description: t(
          "settings.general.workspace.mcp.capability.disabled.description"
        ),
      },
      {
        capability: WorkspaceProfileSetting_MCPCapability.MCP_READ_ONLY,
        label: t("settings.general.workspace.mcp.capability.read-only.self"),
        description: t(
          "settings.general.workspace.mcp.capability.read-only.description"
        ),
      },
      {
        capability: WorkspaceProfileSetting_MCPCapability.MCP_READ_WRITE,
        label: t("settings.general.workspace.mcp.capability.read-write.self"),
        description: t(
          "settings.general.workspace.mcp.capability.read-write.description"
        ),
      },
    ];

    return (
      <FormSection id="mcp" title={title}>
        <PermissionGuard
          permissions={["bb.settings.setWorkspaceProfile"]}
          display="block"
        >
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
                    disabled={!canEdit}
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
