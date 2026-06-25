import {
  Check,
  ChevronDown,
  Copy,
  Link2,
  LockKeyhole,
  Users,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { useCurrentUser } from "@/react/hooks/useAppState";
import { writeTextToClipboard } from "@/react/lib/clipboard";
import { cn } from "@/react/lib/utils";
import { router } from "@/react/router";
import { SQL_EDITOR_WORKSHEET_MODULE } from "@/react/router/handles";
import { useAppStore } from "@/react/stores/app";
import type { Worksheet } from "@/types/proto-es/v1/worksheet_service_pb";
import { Worksheet_Visibility } from "@/types/proto-es/v1/worksheet_service_pb";
import { extractProjectResourceName, extractWorksheetID } from "@/utils";

type AccessOption = {
  label: string;
  description: string;
  value: Worksheet_Visibility;
  icon: React.ReactNode;
};

type Props = {
  readonly worksheet?: Worksheet;
};

/**
 * Replaces frontend/src/views/sql-editor/EditorCommon/SharePopover.vue.
 * Renders the share popover body content: visibility selector + shareable link.
 */
export function SharePopoverBody({ worksheet }: Props) {
  const { t } = useTranslation();
  const workspaceExternalURL = useAppStore((s) => s.serverInfo?.externalUrl);
  const currentUser = useCurrentUser();

  const accessOptions = useMemo<AccessOption[]>(
    () => [
      {
        label: t("sql-editor.private"),
        description: t("sql-editor.private-desc"),
        value: Worksheet_Visibility.PRIVATE,
        icon: <LockKeyhole className="size-5" />,
      },
      {
        label: t("sql-editor.project-read"),
        description: t("sql-editor.project-read-desc"),
        value: Worksheet_Visibility.PROJECT_READ,
        icon: <Users className="size-5" />,
      },
      {
        label: t("sql-editor.project-write"),
        description: t("sql-editor.project-write-desc"),
        value: Worksheet_Visibility.PROJECT_WRITE,
        icon: <Users className="size-5" />,
      },
    ],
    [t]
  );

  const allowChangeAccess = useMemo(() => {
    if (!worksheet || !currentUser) return false;
    return worksheet.creator === `users/${currentUser.email}`;
  }, [worksheet, currentUser]);

  const [currentAccess, setCurrentAccess] = useState<AccessOption>(
    () => accessOptions[0]
  );

  const [selectorOpen, setSelectorOpen] = useState(false);

  // Sync currentAccess from worksheet.visibility when worksheet changes
  useEffect(() => {
    if (!worksheet) return;
    const idx = accessOptions.findIndex(
      (opt) => opt.value === worksheet.visibility
    );
    setCurrentAccess(idx !== -1 ? accessOptions[idx] : accessOptions[0]);
  }, [worksheet, accessOptions]);

  const sharedTabLink = useMemo(() => {
    if (!worksheet) return "";
    const route = router.resolve({
      name: SQL_EDITOR_WORKSHEET_MODULE,
      params: {
        project: extractProjectResourceName(worksheet.project),
        sheet: extractWorksheetID(worksheet.name),
      },
    });
    return new URL(route.href, workspaceExternalURL || window.location.origin)
      .href;
  }, [worksheet, workspaceExternalURL]);

  const handleChangeAccess = async (option: AccessOption) => {
    if (!allowChangeAccess || !worksheet) {
      setSelectorOpen(false);
      return;
    }
    setCurrentAccess(option);
    await useAppStore
      .getState()
      .patchWorksheet({ ...worksheet, visibility: option.value }, [
        "visibility",
      ]);

    if (await writeTextToClipboard(sharedTabLink)) {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-editor.url-copied-to-clipboard"),
      });
    } else {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    }

    // Close only the inner access selector — keep the outer share
    // popover open so the user can still copy the just-updated link
    // (or change the access again).
    setSelectorOpen(false);
  };

  const handleCopyLink = async () => {
    if (await writeTextToClipboard(sharedTabLink)) {
      useAppStore.getState().notify({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-editor.url-copied-to-clipboard"),
      });
    } else {
      // clipboard not available
    }
  };

  return (
    <div className="w-96 p-2 flex flex-col gap-y-4">
      {/* Header: Share title + visibility selector */}
      <section className="w-full flex flex-row justify-between items-center">
        <div className="pr-4">
          <h2 className="text-lg font-semibold">{t("common.share")}</h2>
        </div>
        <Popover open={selectorOpen} onOpenChange={setSelectorOpen}>
          <PopoverTrigger
            // The trigger renders a <div>, not a native <button>; tell Base
            // UI so it doesn't warn about missing native button semantics.
            nativeButton={false}
            render={
              <div
                data-access-trigger
                data-disabled={!allowChangeAccess ? "true" : "false"}
                className={cn(
                  "flex items-center",
                  allowChangeAccess ? "cursor-pointer" : "cursor-not-allowed"
                )}
              />
            }
          >
            <span className="pr-2">{t("sql-editor.link-access")}:</span>
            <div
              className={cn(
                "border flex flex-row justify-start items-center px-2 py-1 rounded-sm",
                allowChangeAccess
                  ? "hover:border-accent"
                  : "border-control-border text-control-placeholder"
              )}
            >
              <strong>{currentAccess.label}</strong>
              <ChevronDown className="size-4 ml-1" />
            </div>
          </PopoverTrigger>
          <PopoverContent side="bottom" align="end" className="w-80 p-2">
            <div className="flex flex-col gap-y-2">
              {accessOptions.map((option) => (
                <div
                  key={option.value}
                  data-option-row
                  className={cn(
                    "p-2 rounded-xs flex justify-between",
                    allowChangeAccess && "cursor-pointer hover:bg-control-bg",
                    option.value === currentAccess.value && "bg-control-bg"
                  )}
                  onClick={() => handleChangeAccess(option)}
                >
                  <div>
                    <div className="flex gap-x-2 items-center">
                      {option.icon}
                      <h2 className="text-md flex">{option.label}</h2>
                    </div>
                    <span className="text-xs textinfolabel">
                      {option.description}
                    </span>
                  </div>
                  {option.value === currentAccess.value && (
                    <div className="flex items-center">
                      <Check className="size-5" />
                    </div>
                  )}
                </div>
              ))}
            </div>
          </PopoverContent>
        </Popover>
      </section>

      {/* Link input + copy button — single bordered container with rounded
          inner corners. No group-level focus ring; only the input shows a
          focus highlight. */}
      <div className="flex items-center h-8 rounded-xs border border-control-border overflow-hidden">
        {/* Link icon prefix (gray addon) */}
        <div className="flex items-center justify-center h-full px-2 bg-control-bg text-control-light border-r border-control-border">
          <Link2 className="size-5" />
        </div>
        {/* URL input — always read-only; the link itself is not editable. */}
        <input
          type="text"
          readOnly
          value={sharedTabLink}
          className="flex-1 min-w-0 h-full px-2 bg-background text-control text-sm cursor-text appearance-none border-0 shadow-none outline-hidden focus:outline-hidden focus:ring-0 focus:border-0 focus:shadow-none"
        />
        {/* Copy button — enabled whenever the shared worksheet has a link.
            Gated on the shared worksheet (sharedTabLink), NOT the current tab's
            status: the popover can be opened for any worksheet from the tree,
            so the current tab's dirty state is irrelevant. Only an unsaved draft
            (no worksheet → no link) disables it. */}
        <Button
          type="button"
          variant="ghost"
          size="sm"
          data-copy-btn
          disabled={!sharedTabLink}
          onClick={handleCopyLink}
          className="h-full rounded-none border-l border-control-border bg-background enabled:hover:bg-control-bg-hover enabled:hover:text-main disabled:bg-control-bg focus-visible:ring-inset focus-visible:ring-offset-0"
        >
          <Copy className="size-4" />
        </Button>
      </div>
    </div>
  );
}
