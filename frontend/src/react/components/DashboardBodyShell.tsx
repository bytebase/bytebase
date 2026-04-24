import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { useTranslation } from "react-i18next";
import { DashboardHeader } from "@/react/components/header/DashboardHeader";
import type {
  DashboardBodyShellProps,
  DashboardShellTargets,
} from "@/react/dashboard-shell";
import { cn } from "@/react/lib/utils";

function subscribeToViewport(onStoreChange: () => void) {
  window.addEventListener("resize", onStoreChange);
  return () => window.removeEventListener("resize", onStoreChange);
}

function getDesktopSnapshot() {
  return window.innerWidth >= 768;
}

function getServerDesktopSnapshot() {
  return true;
}

function useDesktopViewport() {
  return useSyncExternalStore(
    subscribeToViewport,
    getDesktopSnapshot,
    getServerDesktopSnapshot
  );
}

export function DashboardBodyShell({
  variant,
  isRootPath = false,
  routeKey = "",
  onReady,
}: DashboardBodyShellProps) {
  const { t } = useTranslation();
  const desktopSidebarRef = useRef<HTMLDivElement>(null);
  const mobileSidebarRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const quickstartRef = useRef<HTMLDivElement>(null);
  const mainContainerRef = useRef<HTMLDivElement>(null);
  const isDesktop = useDesktopViewport();
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false);

  const showWorkspaceChrome = variant === "workspace" && !isRootPath;
  const showHeader = variant === "issues" || showWorkspaceChrome;

  useEffect(() => {
    setIsMobileSidebarOpen(false);
  }, [routeKey]);

  useEffect(() => {
    if (isDesktop) {
      setIsMobileSidebarOpen(false);
    }
  }, [isDesktop]);

  useEffect(() => {
    const targets: DashboardShellTargets = {
      desktopSidebar: showWorkspaceChrome ? desktopSidebarRef.current : null,
      mobileSidebar: showWorkspaceChrome ? mobileSidebarRef.current : null,
      content: contentRef.current,
      quickstart: quickstartRef.current,
      mainContainer: mainContainerRef.current,
    };
    onReady?.(targets);
  }, [isDesktop, onReady, showHeader, showWorkspaceChrome, variant]);

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex flex-1 overflow-hidden">
        {showWorkspaceChrome ? (
          <>
            <div
              className={cn(
                "fixed inset-0 z-40 md:hidden",
                isMobileSidebarOpen ? "" : "pointer-events-none"
              )}
            >
              <button
                aria-label={t("common.close-mobile-sidebar")}
                className={cn(
                  "absolute inset-0 bg-black/20 transition-opacity",
                  isMobileSidebarOpen ? "opacity-100" : "opacity-0"
                )}
                type="button"
                onClick={() => setIsMobileSidebarOpen(false)}
              />
              <div
                className={cn(
                  "absolute inset-y-0 left-0 w-52 bg-control-bg transition-transform",
                  isMobileSidebarOpen ? "translate-x-0" : "-translate-x-full"
                )}
              >
                <div
                  ref={mobileSidebarRef}
                  className="h-full overflow-y-auto bg-control-bg"
                />
              </div>
            </div>

            <aside
              className={cn("shrink-0", isDesktop ? "flex" : "hidden")}
              data-label="bb-dashboard-static-sidebar"
            >
              <div className="flex w-52 flex-col bg-control-bg">
                <div
                  ref={desktopSidebarRef}
                  className="flex flex-1 flex-col overflow-y-auto py-0"
                />
              </div>
            </aside>
          </>
        ) : null}

        <div
          className={cn(
            "flex min-w-0 flex-1 flex-col",
            variant === "issues" ? "border-x border-block-border" : ""
          )}
          data-label="bb-main-body-wrapper"
        >
          {showHeader ? (
            <nav
              className="border-b border-block-border bg-white"
              data-label="bb-dashboard-header"
            >
              <div className="mx-auto max-w-full">
                <DashboardHeader
                  showLogo={variant === "issues"}
                  showMobileSidebarToggle={showWorkspaceChrome}
                  onOpenMobileSidebar={() => setIsMobileSidebarOpen(true)}
                />
              </div>
            </nav>
          ) : null}

          <div
            id="bb-layout-main"
            ref={mainContainerRef}
            className={cn(
              "flex-1 overflow-y-auto",
              variant === "workspace" ? "md:min-w-0" : "min-w-0"
            )}
          >
            <div ref={contentRef} />
          </div>
        </div>
      </div>

      <div ref={quickstartRef} />
    </div>
  );
}
