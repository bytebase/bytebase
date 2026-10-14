import { ArrowDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import accessControlsImage from "@/assets/illustration/setup-access-controls.webp";
import exploreImage from "@/assets/illustration/setup-explore.webp";
import heroImage from "@/assets/illustration/setup-hero.webp";
import queryImage from "@/assets/illustration/setup-query.webp";
import logoFull from "@/assets/logo-full.svg";
import { UserAvatar } from "@/components/UserAvatar";
import { cn } from "@/lib/utils";
import { PresetRoleType } from "@/types/iam/role";
import { displayRoleTitle } from "@/utils/role";
import type { SetupGoal, SetupScreen, SetupSource } from "./draft";

// Each still with its pixel size, which holds the picture's place in the
// panel until the file has loaded.
const STILLS = {
  hero: { src: heroImage, width: 1222, height: 829 },
  query: { src: queryImage, width: 1280, height: 887 },
  explore: { src: exploreImage, width: 1280, height: 782 },
  "access-controls": { src: accessControlsImage, width: 1368, height: 1192 },
} as const;

type Still = keyof typeof STILLS;

// The chosen goal's picture, then who gets access. Until a goal is chosen the
// workspace preview stays.
const stillFor = (
  screen: "goal" | "team",
  goal: SetupGoal | undefined
): Still | undefined => {
  if (screen === "team") return "access-controls";
  switch (goal) {
    case "create-database-change":
      return "hero";
    case "query-data":
      return "query";
    case "just-exploring":
      return "explore";
    case undefined:
      return undefined;
  }
};

const FRAME =
  "w-full max-w-lg overflow-hidden rounded-sm border border-block-border";

// The workspace as screen 1 names it, with its first member: what the two
// names are for.
function WorkspacePreview({
  workspaceTitle,
  memberName,
  memberEmail,
}: {
  readonly workspaceTitle: string;
  readonly memberName: string;
  readonly memberEmail: string;
}) {
  return (
    <div className={cn(FRAME, "bg-background")} aria-hidden="true">
      <div className="bg-accent/5 p-7">
        <div className="overflow-hidden rounded-sm border border-block-border bg-background">
          <div className="flex h-11 items-center gap-x-2 border-b border-block-border px-3.5 text-sm text-main">
            <img src={logoFull} className="h-4 shrink-0" alt="" />
            <span className="text-control-placeholder">/</span>
            <span className="truncate">{workspaceTitle}</span>
            <UserAvatar
              title={memberName}
              colorSeed={memberEmail}
              size="sm"
              className="ml-auto"
            />
          </div>
          <div className="flex h-64">
            <div className="flex w-28 shrink-0 flex-col gap-y-3 border-r border-block-border p-3">
              {[70, 55, 80, 60, 75, 50, 65].map((width, index) => (
                <div
                  key={index}
                  className="h-2 rounded-full bg-control-bg"
                  style={{ width: `${width}%` }}
                />
              ))}
            </div>
            <div className="min-w-0 flex-1 p-4">
              <div className="truncate font-medium text-main">
                {workspaceTitle}
              </div>
              <div className="mt-3 divide-y divide-block-border border-t border-block-border">
                <div className="flex items-center gap-x-2 py-2.5 text-sm text-main">
                  <UserAvatar
                    title={memberName}
                    colorSeed={memberEmail}
                    size="sm"
                  />
                  <span className="max-w-[60%] shrink-0 truncate">
                    {memberName}
                  </span>
                  <span className="ml-auto min-w-0 truncate text-xs text-control-light">
                    {displayRoleTitle(PresetRoleType.WORKSPACE_ADMIN)}
                  </span>
                </div>
                {[24, 18].map((width, index) => (
                  <div
                    key={index}
                    className="flex items-center gap-x-2 py-2.5 opacity-40"
                  >
                    <div className="size-7 shrink-0 rounded-full bg-control-bg-hover" />
                    <div
                      className="h-2 rounded-full bg-control-bg-hover"
                      style={{ width: `${width}%` }}
                    />
                    <span className="ml-auto min-w-0 truncate text-xs text-control-light">
                      {displayRoleTitle(PresetRoleType.WORKSPACE_MEMBER)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function DiagramBox({
  title,
  detail,
  pending = false,
  accent = false,
}: {
  readonly title: string;
  readonly detail: string;
  // Not there yet: it arrives after setup.
  readonly pending?: boolean;
  readonly accent?: boolean;
}) {
  return (
    <div
      className={cn(
        "w-full rounded-sm border bg-background px-4 py-3",
        accent ? "border-accent" : "border-block-border",
        pending && "border-dashed"
      )}
    >
      <div className="truncate font-medium text-main">{title}</div>
      <div className="mt-0.5 text-sm text-control-light">{detail}</div>
    </div>
  );
}

function DiagramArrow({ label }: { readonly label: string }) {
  return (
    <div className="flex items-center gap-x-1 text-xs text-control-light">
      <ArrowDown className="size-3.5" aria-hidden="true" />
      {label}
    </div>
  );
}

export function SetupPanel({
  screen,
  goal,
  workspaceTitle,
  memberName,
  memberEmail,
  projectTitle,
  source,
  isSaaS,
}: {
  readonly screen: SetupScreen;
  readonly goal: SetupGoal | undefined;
  readonly workspaceTitle: string;
  readonly memberName: string;
  readonly memberEmail: string;
  readonly projectTitle: string;
  readonly source: SetupSource;
  readonly isSaaS: boolean;
}) {
  const { t } = useTranslation();

  const still =
    screen === "goal" || screen === "team" ? stillFor(screen, goal) : undefined;
  if (screen === "workspace" || (screen === "goal" && !still)) {
    return (
      <WorkspacePreview
        workspaceTitle={workspaceTitle}
        memberName={memberName}
        memberEmail={memberEmail}
      />
    );
  }
  if (still) {
    return (
      <div className={cn(FRAME, "bg-background")}>
        {/* No taller than the window, so changing pictures does not move the
            form beside it. */}
        <img
          className="block h-auto max-h-[calc(100vh-6rem)] w-full object-contain"
          {...STILLS[still]}
          data-still={still}
          alt=""
        />
      </div>
    );
  }

  return (
    <div className="flex w-full max-w-sm flex-col items-center gap-y-2">
      {source === "sample" ? (
        <>
          <DiagramBox
            title={t("workspace-setup.panel.sample-instance")}
            detail={
              isSaaS
                ? t("workspace-setup.panel.sample-instance-detail")
                : t("workspace-setup.panel.sample-instance-detail-self-host")
            }
          />
          <DiagramArrow label={t("workspace-setup.panel.holds")} />
          <DiagramBox
            title={t("workspace-setup.panel.sample-database")}
            detail={t("workspace-setup.panel.sample-database-detail")}
          />
        </>
      ) : (
        <>
          <DiagramBox
            pending
            title={t("workspace-setup.panel.own-instance")}
            detail={t("workspace-setup.panel.own-instance-detail")}
          />
          <DiagramArrow label={t("workspace-setup.panel.holds")} />
          <DiagramBox
            pending
            title={t("workspace-setup.panel.own-databases")}
            detail={t("workspace-setup.panel.own-databases-detail")}
          />
        </>
      )}
      <DiagramArrow label={t("workspace-setup.panel.assigned-to")} />
      <DiagramBox
        accent
        title={projectTitle || t("workspace-setup.project.your-project")}
        detail={t("common.project")}
      />
    </div>
  );
}
