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
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/react/components/ui/popover";
import { useVueState } from "@/react/hooks/useVueState";
import { cn } from "@/react/lib/utils";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
import { router } from "@/router";
import { SQL_EDITOR_WORKSHEET_MODULE } from "@/router/sqlEditor";
import {
  pushNotification,
  useActuatorV1Store,
  useCurrentUserV1,
  useWorkSheetStore,
} from "@/store";
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
  const actuatorStore = useActuatorV1Store();
  const currentUserStore = useCurrentUserV1();
  const worksheetStore = useWorkSheetStore();
  const tabStore = useSQLEditorTabStore();

  const workspaceExternalURL = useVueState(
    () => actuatorStore.serverInfo?.externalUrl
  );
  const currentUser = useVueState(() => currentUserStore.value);
  const tabStatus = useVueState(() => tabStore.currentTab?.status);

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
    await worksheetStore.patchWorksheet(
      { ...worksheet, visibility: option.value },
      ["visibility"]
    );

    try {
      await navigator.clipboard.writeText(sharedTabLink);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-editor.url-copied-to-clipboard"),
      });
    } catch {
      pushNotification({
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
    try {
      await navigator.clipboard.writeText(sharedTabLink);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("sql-editor.url-copied-to-clipboard"),
      });
    } catch {
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
                  : "border-gray-200 text-gray-400"
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
                    allowChangeAccess && "cursor-pointer hover:bg-gray-200",
                    option.value === currentAccess.value && "bg-gray-200"
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

      {/* Link input + copy button */}
      <div className="flex items-center gap-x-0">
        {/* Link icon prefix */}
        <div className="flex items-center justify-center px-2 py-1 border border-r-0 border-control-border bg-control-bg rounded-l-xs h-8">
          <Link2 className="size-5" />
        </div>
        {/* URL input */}
        <input
          type="text"
          readOnly
          value={sharedTabLink}
          className="flex-1 min-w-0 px-2 py-1 border border-control-border bg-control-bg text-control text-sm h-8 focus:outline-none"
        />
        {/* Copy button */}
        <button
          type="button"
          data-copy-btn
          disabled={tabStatus !== "CLEAN"}
          onClick={handleCopyLink}
          className="flex items-center justify-center px-2 py-1 border border-l-0 border-control-border bg-control-bg rounded-r-xs h-8 text-control-light hover:text-main disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Copy className="size-4" />
        </button>
      </div>
    </div>
  );
}
