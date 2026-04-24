export type DashboardFrameShellTargets = {
  banner: HTMLDivElement | null;
  body: HTMLDivElement | null;
};

export type DashboardShellTargets = {
  desktopSidebar: HTMLDivElement | null;
  mobileSidebar: HTMLDivElement | null;
  content: HTMLDivElement | null;
  quickstart: HTMLDivElement | null;
  mainContainer: HTMLDivElement | null;
};

export type DashboardBodyShellVariant = "workspace" | "issues";

export type DashboardFrameShellProps = {
  onReady?: (targets: DashboardFrameShellTargets) => void;
};

export type DashboardBodyShellProps = {
  variant: DashboardBodyShellVariant;
  isRootPath?: boolean;
  routeKey?: string;
  onReady?: (targets: DashboardShellTargets) => void;
};
