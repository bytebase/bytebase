import { X } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/react/components/ui/button";
import { cn } from "@/react/lib/utils";
import { IssueDetailActivity } from "./issue-detail/components/IssueDetailActivity";
import { IssueDetailBranchContent } from "./issue-detail/components/IssueDetailBranchContent";
import { IssueDetailHeader } from "./issue-detail/components/IssueDetailHeader";
import { IssueDetailSidebar } from "./issue-detail/components/IssueDetailSidebar";
import { IssueDetailProvider } from "./issue-detail/context/IssueDetailContext";
import { useIssueDetailPage } from "./issue-detail/hooks/useIssueDetailPage";
import type { ProjectIssueDetailPageProps } from "./issue-detail/types";

export function ProjectIssueDetailPage(props: ProjectIssueDetailPageProps) {
  const { t } = useTranslation();
  const [
    databaseExportExecutionHistoryExpanded,
    setDatabaseExportExecutionHistoryExpanded,
  ] = useState(false);
  const [databaseExportTasksExpanded, setDatabaseExportTasksExpanded] =
    useState(false);
  const [pageHost, setPageHost] = useState<HTMLDivElement | null>(null);
  const [databaseChangeSelectedSpecId, setDatabaseChangeSelectedSpecId] =
    useState("");
  const page = useIssueDetailPage({ ...props, pageHost });
  const showDesktopSidebar = page.sidebarMode === "DESKTOP";
  const showMobileSidebar = page.sidebarMode === "MOBILE";
  const desktopSidebarStyle = useMemo(
    () => ({
      width: `${page.desktopSidebarWidth || 240}px`,
    }),
    [page.desktopSidebarWidth]
  );

  return (
    <IssueDetailProvider value={page}>
      <div
        ref={setPageHost}
        className="relative h-full overflow-x-hidden overflow-y-auto"
      >
        <div
          className={`flex min-h-full flex-col ${
            page.ready ? "" : "invisible pointer-events-none"
          }`}
        >
          <IssueDetailHeader />
          <div className="flex flex-1 items-stretch border-t">
            <div className="flex flex-1 shrink flex-col gap-y-4 overflow-x-auto p-4">
              <IssueDetailBranchContent
                databaseChangeSelectedSpecId={databaseChangeSelectedSpecId}
                databaseExportExecutionHistoryExpanded={
                  databaseExportExecutionHistoryExpanded
                }
                databaseExportTasksExpanded={databaseExportTasksExpanded}
                onDatabaseExportExecutionHistoryExpandedChange={
                  setDatabaseExportExecutionHistoryExpanded
                }
                onDatabaseChangeSelectedSpecIdChange={
                  setDatabaseChangeSelectedSpecId
                }
                onDatabaseExportTasksExpandedChange={
                  setDatabaseExportTasksExpanded
                }
              />
              <IssueDetailActivity />
            </div>

            {showDesktopSidebar && (
              <div
                className="shrink-0 border-l bg-white"
                style={desktopSidebarStyle}
              >
                <div className="sticky top-0">
                  <IssueDetailSidebar />
                </div>
              </div>
            )}
          </div>
        </div>

        {!page.ready && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-y-3 bg-white">
            <div className="h-8 w-8 animate-spin rounded-full border-2 border-control-border border-t-accent" />
            <div className="text-sm text-control-light">
              {t("common.loading")}
            </div>
          </div>
        )}

        {showMobileSidebar && (
          <div
            className={cn(
              "absolute inset-0 z-30 transition-[visibility] duration-200",
              page.mobileSidebarOpen
                ? "visible"
                : "invisible pointer-events-none"
            )}
          >
            <button
              className={cn(
                "absolute inset-0 bg-black/40 transition-opacity duration-200",
                page.mobileSidebarOpen ? "opacity-100" : "opacity-0"
              )}
              onClick={() => page.setMobileSidebarOpen(false)}
              type="button"
            />
            <div
              className={cn(
                "absolute right-0 top-0 flex h-full w-[80vw] min-w-[240px] max-w-[320px] transform flex-col border-l bg-white p-2 shadow-lg transition-transform duration-200",
                page.mobileSidebarOpen ? "translate-x-0" : "translate-x-full"
              )}
            >
              <div className="mb-2 flex justify-end">
                <Button
                  aria-label={t("common.close")}
                  onClick={() => page.setMobileSidebarOpen(false)}
                  size="sm"
                  variant="ghost"
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto">
                <IssueDetailSidebar />
              </div>
            </div>
          </div>
        )}
      </div>
    </IssueDetailProvider>
  );
}
