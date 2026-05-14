import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { aiContextEvents } from "@/plugins/ai/logic";
import * as promptUtils from "@/plugins/ai/logic/prompt";
import type { ChatAction } from "@/plugins/ai/types";
import { Button } from "@/react/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/react/components/ui/dropdown-menu";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { router } from "@/router";
import { SETTING_ROUTE_WORKSPACE_GENERAL } from "@/router/dashboard/workspaceSetting";
import {
  useConnectionOfCurrentSQLEditorTab,
  useSettingV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { hasWorkspacePermissionV2, nextAnimationFrame } from "@/utils";

type Size = "sm" | "default";

type Props = {
  /**
   * Restricts the popselect to the provided subset of actions. When omitted
   * all built-in actions (explain-code, find-problems) render.
   */
  readonly actions?: ChatAction[];
  /** The SQL statement the assistant should reason about. */
  readonly statement?: string;
  readonly size?: Size;
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/OpenAIButton/*.vue.
 * Shows an AI-assistant toggle when the editor is connected + in worksheet
 * mode. Clicking the button toggles the AI panel; the attached dropdown
 * offers shortcuts that seed the chat with a prompt.
 */
export function OpenAIButton({ actions, statement, size = "default" }: Props) {
  const { t } = useTranslation();
  const tabStore = useSQLEditorTabStore();
  const showAIPanel = useSQLEditorStore((s) => s.showAIPanel);
  const setShowAIPanel = useSQLEditorStore((s) => s.setShowAIPanel);
  const settingV1Store = useSettingV1Store();
  const { instance: instanceRef } = useConnectionOfCurrentSQLEditorTab();

  // Make sure the AI setting is resolved before we gate the enabled state.
  useEffect(() => {
    void settingV1Store.getOrFetchSettingByName(
      Setting_SettingName.AI,
      true /* silent */
    );
  }, [settingV1Store]);

  const isDisconnected = useVueState(() => tabStore.isDisconnected);
  const currentMode = useVueState(() => tabStore.currentTab?.mode);
  const instance = useVueState(() => instanceRef.value);
  const openAIEnabled = useVueState(() => {
    const setting = settingV1Store.getSettingByName(Setting_SettingName.AI);
    return setting?.value?.value?.case === "ai"
      ? setting.value.value.value.enabled
      : false;
  });

  const [menuOpen, setMenuOpen] = useState(false);

  if (isDisconnected || currentMode !== "WORKSHEET") {
    return null;
  }

  const buttonVariant = showAIPanel ? "default" : "outline";

  const icon = (
    <svg
      viewBox="0 0 320 320"
      fill="currentColor"
      stroke="none"
      className="size-4"
      aria-hidden="true"
    >
      <path d="m297.06 130.97c7.26-21.79 4.76-45.66-6.85-65.48-17.46-30.4-52.56-46.04-86.84-38.68-15.25-17.18-37.16-26.95-60.13-26.81-35.04-.08-66.13 22.48-76.91 55.82-22.51 4.61-41.94 18.7-53.31 38.67-17.59 30.32-13.58 68.54 9.92 94.54-7.26 21.79-4.76 45.66 6.85 65.48 17.46 30.4 52.56 46.04 86.84 38.68 15.24 17.18 37.16 26.95 60.13 26.8 35.06.09 66.16-22.49 76.94-55.86 22.51-4.61 41.94-18.7 53.31-38.67 17.57-30.32 13.55-68.51-9.94-94.51zm-120.28 168.11c-14.03.02-27.62-4.89-38.39-13.88.49-.26 1.34-.73 1.89-1.07l63.72-36.8c3.26-1.85 5.26-5.32 5.24-9.07v-89.83l26.93 15.55c.29.14.48.42.52.74v74.39c-.04 33.08-26.83 59.9-59.91 59.97zm-128.84-55.03c-7.03-12.14-9.56-26.37-7.15-40.18.47.28 1.3.79 1.89 1.13l63.72 36.8c3.23 1.89 7.23 1.89 10.47 0l77.79-44.92v31.1c.02.32-.13.63-.38.83l-64.41 37.19c-28.69 16.52-65.33 6.7-81.92-21.95zm-16.77-139.09c7-12.16 18.05-21.46 31.21-26.29 0 .55-.03 1.52-.03 2.2v73.61c-.02 3.74 1.98 7.21 5.23 9.06l77.79 44.91-26.93 15.55c-.27.18-.61.21-.91.08l-64.42-37.22c-28.63-16.58-38.45-53.21-21.95-81.89zm221.26 51.49-77.79-44.92 26.93-15.54c.27-.18.61-.21.91-.08l64.42 37.19c28.68 16.57 38.51 53.26 21.94 81.94-7.01 12.14-18.05 21.44-31.2 26.28v-75.81c.03-3.74-1.96-7.2-5.2-9.06zm26.8-40.34c-.47-.29-1.3-.79-1.89-1.13l-63.72-36.8c-3.23-1.89-7.23-1.89-10.47 0l-77.79 44.92v-31.1c-.02-.32.13-.63.38-.83l64.41-37.16c28.69-16.55 65.37-6.7 81.91 22 6.99 12.12 9.52 26.31 7.15 40.1zm-168.51 55.43-26.94-15.55c-.29-.14-.48-.42-.52-.74v-74.39c.02-33.12 26.89-59.96 60.01-59.94 14.01 0 27.57 4.92 38.34 13.88-.49.26-1.33.73-1.89 1.07l-63.72 36.8c-3.26 1.85-5.26 5.31-5.24 9.06l-.04 89.79zm14.63-31.54 34.65-20.01 34.65 20v40.01l-34.65 20-34.65-20z" />
    </svg>
  );

  const buttonClass = cn(
    "h-7 px-1.5 gap-1",
    showAIPanel && "bg-accent/10 text-accent hover:bg-accent/20"
  );

  const handleToggle = () => {
    setShowAIPanel(!showAIPanel);
  };

  // ---- Disabled state: AI feature not configured ---------------------------
  if (!openAIEnabled) {
    return (
      <AINotConfiguredButton size={size} icon={icon} className={buttonClass} />
    );
  }

  // ---- Enabled state -------------------------------------------------------
  const allActions: { value: ChatAction; label: string }[] = [
    { value: "explain-code", label: t("plugin.ai.actions.explain-code") },
    { value: "find-problems", label: t("plugin.ai.actions.find-problems") },
  ];
  const visibleActions = actions
    ? allActions.filter((o) => actions.includes(o.value))
    : allActions;
  const hasActions = visibleActions.length > 0;

  const handleSelect = async (action: ChatAction) => {
    setMenuOpen(false);
    const newChat = !showAIPanel;
    setShowAIPanel(true);

    if (!statement) return;
    await nextAnimationFrame();
    if (action === "explain-code") {
      void aiContextEvents.emit("send-chat", {
        content: promptUtils.explainCode(statement, instance.engine),
        newChat,
      });
    } else if (action === "find-problems") {
      void aiContextEvents.emit("send-chat", {
        content: promptUtils.findProblems(statement, instance.engine),
        newChat,
      });
    }
  };

  const renderButton = (onClick?: () => void) => (
    <Button
      variant={buttonVariant}
      size={size}
      className={buttonClass}
      aria-label={t("plugin.ai.ai-assistant")}
      onClick={onClick}
    >
      {icon}
    </Button>
  );

  if (!hasActions) {
    return renderButton(handleToggle);
  }

  return (
    <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
      <DropdownMenuTrigger
        render={renderButton(handleToggle)}
        onContextMenu={(e) => {
          // Right-click opens the action menu; left-click is handled by the
          // button's onClick and toggles the AI panel.
          e.preventDefault();
          setMenuOpen(true);
        }}
      />
      <DropdownMenuContent align="end" sideOffset={4}>
        <div className="px-3 py-1.5 text-xs font-semibold text-control-light">
          {t("plugin.ai.ai-assistant")}
        </div>
        {visibleActions.map((opt) => (
          <DropdownMenuItem
            key={opt.value}
            disabled={!statement}
            onClick={() => {
              void handleSelect(opt.value);
            }}
          >
            {opt.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** Hover-triggered popover shown when AI is not configured. */
function AINotConfiguredButton({
  size,
  icon,
  className,
}: {
  size: "sm" | "default";
  icon: React.ReactNode;
  className: string;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const allowConfigure = hasWorkspacePermissionV2("bb.settings.set");

  const handleEnter = useCallback(() => {
    if (closeTimerRef.current !== null) clearTimeout(closeTimerRef.current);
    setOpen(true);
  }, []);

  const handleLeave = useCallback(() => {
    closeTimerRef.current = setTimeout(() => setOpen(false), 150);
  }, []);

  useEffect(
    () => () => {
      if (closeTimerRef.current !== null) clearTimeout(closeTimerRef.current);
    },
    []
  );

  return (
    <Popover open={open}>
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            size={size}
            aria-disabled
            className={cn(className, "opacity-50 cursor-default")}
            aria-label={t("plugin.ai.ai-assistant")}
            onMouseEnter={handleEnter}
            onMouseLeave={handleLeave}
          >
            {icon}
          </Button>
        }
      />
      <PopoverContent
        align="end"
        className="max-w-[20rem]"
        onMouseEnter={handleEnter}
        onMouseLeave={handleLeave}
      >
        <div className="flex flex-col">
          <div className="border-b pb-1 font-semibold">
            {t("plugin.ai.ai-assistant")}
          </div>
          <div className="pt-2 flex flex-col text-control-light">
            <p>
              {t("plugin.ai.not-configured.self")}{" "}
              {allowConfigure ? (
                <Button
                  variant="link"
                  size="sm"
                  tabIndex={-1}
                  className="h-auto p-0 text-accent"
                  onClick={() => {
                    setOpen(false);
                    void router.push({
                      name: SETTING_ROUTE_WORKSPACE_GENERAL,
                      hash: "#ai-assistant",
                    });
                  }}
                >
                  {t("plugin.ai.not-configured.go-to-configure")}
                </Button>
              ) : (
                t("plugin.ai.not-configured.contact-admin-to-configure")
              )}
            </p>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
