import { Database, Folder } from "lucide-react";
import { type ReactNode, useLayoutEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import exploreImage from "@/assets/illustration/setup-explore.webp";
import heroImage from "@/assets/illustration/setup-hero.webp";
import queryImage from "@/assets/illustration/setup-query.webp";
import logoFull from "@/assets/logo-full.svg";
import { EngineIcon } from "@/components/EngineIcon";
import { UserAvatar } from "@/components/UserAvatar";
import { cn } from "@/lib/utils";
import type { GuideTeamSize } from "@/modules/workspace-setup-guide/types";
import { PresetRoleType } from "@/types/iam/role";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { displayRoleTitle } from "@/utils/role";
import type { SetupGoal, SetupScreen, SetupSource } from "./draft";

// Each still with its pixel size, which holds the picture's place in the
// panel until the file has loaded.
const STILLS = {
  hero: { src: heroImage, width: 1222, height: 829 },
  query: { src: queryImage, width: 1280, height: 887 },
  explore: { src: exploreImage, width: 1280, height: 782 },
} as const;

type Still = keyof typeof STILLS;

const stillFor = (goal: SetupGoal | undefined): Still | undefined => {
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

const FRAME = "overflow-hidden rounded-sm border border-block-border";

// Every picture takes 70% of the panel's column, which is half the window,
// never under 512px or over 800px, and never taller than the window.
const pictureWidth = (width: number, height: number) =>
  `min(100%, clamp(32rem, 35vw, 50rem), calc((100dvh - 6rem) * ${width / height}))`;

const CAPTION = "font-mono text-xs uppercase tracking-wider text-control-light";

// A picture drawn at a fixed size and scaled as one, so a wide screen gets the
// same picture larger.
function Drawn({
  width,
  height,
  children,
}: {
  readonly width: number;
  readonly height: number;
  readonly children: ReactNode;
}) {
  const frameRef = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(1);
  useLayoutEffect(() => {
    const frame = frameRef.current;
    if (!frame) return;
    const measure = () => setScale(frame.clientWidth / width || 1);
    measure();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(measure);
    observer.observe(frame);
    return () => observer.disconnect();
  }, [width]);

  return (
    <div
      ref={frameRef}
      aria-hidden="true"
      className={cn(FRAME, "relative bg-background")}
      style={{
        width: pictureWidth(width, height),
        aspectRatio: `${width} / ${height}`,
      }}
    >
      <div
        className="absolute top-0 left-0 origin-top-left bg-accent/5"
        style={{ width, height, transform: `scale(${scale})` }}
      >
        {children}
      </div>
    </div>
  );
}

function GhostBar({ width }: { readonly width: string }) {
  return (
    <div className="h-2 rounded-full bg-control-bg-hover" style={{ width }} />
  );
}

// At most this many rows of people, the person setting up included.
const MEMBER_ROWS = 5;

// The workspace as screen 1 names it, with its first member: what the two
// names are for. On screen 4 its member rows hold the team: each address
// listed, then, while the team is bigger, dimmed rows.
function WorkspacePreview({
  workspaceTitle,
  memberName,
  memberEmail,
  team,
}: {
  readonly workspaceTitle: string;
  readonly memberName: string;
  readonly memberEmail: string;
  readonly team?: {
    readonly size: GuideTeamSize | undefined;
    readonly invitees: readonly string[];
  };
}) {
  const { t } = useTranslation();
  const shown = team?.invitees.slice(0, MEMBER_ROWS - 1) ?? [];
  const hidden = (team?.invitees.length ?? 0) - shown.length;
  const ghosts = !team
    ? 2
    : team.size && team.size !== "just-me"
      ? MEMBER_ROWS - 1 - shown.length
      : 0;
  const memberRole = displayRoleTitle(PresetRoleType.WORKSPACE_MEMBER);
  return (
    <Drawn width={512} height={team ? 456 : 360}>
      <div className="h-full p-7">
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
          <div className={cn("flex", team ? "h-[352px]" : "h-64")}>
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
                {shown.map((email) => (
                  <div
                    key={email}
                    className="flex items-center gap-x-2 py-2.5 text-sm text-main"
                  >
                    <UserAvatar title={email} colorSeed={email} size="sm" />
                    <span className="min-w-0 truncate">{email}</span>
                    <span className="ml-auto shrink-0 text-xs text-control-light">
                      {memberRole}
                    </span>
                  </div>
                ))}
                {Array.from({ length: ghosts }, (_, index) => (
                  <div
                    key={index}
                    className="flex items-center gap-x-2 py-2.5 opacity-40"
                  >
                    <div className="size-7 shrink-0 rounded-full bg-control-bg-hover" />
                    <GhostBar width={`${[24, 18, 22, 16][index]}%`} />
                    <span className="ml-auto min-w-0 truncate text-xs text-control-light">
                      {memberRole}
                    </span>
                  </div>
                ))}
                {hidden > 0 && (
                  <div className="py-2.5 pl-9 text-xs text-control-light">
                    {t("common.n-more", { n: hidden })}
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </Drawn>
  );
}

function ProjectIcon({ className }: { readonly className?: string }) {
  return (
    <div
      className={cn(
        "flex shrink-0 items-center justify-center rounded-sm bg-accent text-accent-text",
        className
      )}
    >
      <Folder className="size-1/2" />
    </div>
  );
}

function GhostInstance({ left }: { readonly left: number }) {
  return (
    <div
      className="absolute top-[262px] h-[134px] w-[154px] rounded-sm border border-block-border bg-background p-3 opacity-50"
      style={{ left }}
    >
      <div className="flex gap-x-2">
        <div className="size-5 shrink-0 rounded-xs bg-control-bg-hover" />
        <div className="flex flex-1 flex-col gap-y-2 pt-1">
          <GhostBar width="80%" />
          <GhostBar width="45%" />
        </div>
      </div>
      <div className="mt-8 h-8 rounded-xs bg-control-bg-hover" />
    </div>
  );
}

// The workspace holding the project as it is named, and the project's
// instances: the one the database choice brings, then two to come. A project
// left for later is dashed, with no instances.
function ProjectPreview({
  workspaceTitle,
  projectTitle,
  source,
  noProject,
}: {
  readonly workspaceTitle: string;
  readonly projectTitle: string;
  readonly source: SetupSource;
  readonly noProject: boolean;
}) {
  const { t } = useTranslation();
  const own = source === "own";
  return (
    <Drawn width={640} height={452}>
      <div className="absolute inset-6 rounded-sm border border-control-border bg-background/60" />
      <div className="absolute top-10 left-11 max-w-[540px]">
        <div className={CAPTION}>{t("workspace-setup.panel.workspace")}</div>
        <div className="mt-1 truncate text-lg font-semibold text-main">
          {workspaceTitle}
        </div>
      </div>
      <div
        className={cn(
          "absolute top-[108px] left-11 flex h-[76px] w-[296px] items-center gap-x-3 rounded-sm border bg-background px-4",
          noProject
            ? "border-dashed border-control-border text-control-light"
            : "border-block-border text-main"
        )}
      >
        {noProject ? (
          <div className="size-9 shrink-0 rounded-sm bg-control-bg-hover" />
        ) : (
          <ProjectIcon className="size-9" />
        )}
        <div className="min-w-0">
          <div className="truncate text-lg font-semibold">
            {projectTitle || t("workspace-setup.project.your-project")}
          </div>
          <div className={CAPTION}>{t("common.project")}</div>
        </div>
      </div>
      <div className="absolute top-[108px] left-[356px] flex h-[76px] w-[240px] items-center gap-x-3 rounded-sm border border-block-border bg-background px-4 opacity-50">
        <div className="size-9 shrink-0 rounded-sm bg-control-bg-hover" />
        <div className="flex flex-1 flex-col gap-y-2">
          <GhostBar width="70%" />
          <GhostBar width="40%" />
        </div>
      </div>
      {!noProject && (
        <>
          <svg
            className="absolute inset-0 fill-none stroke-control-border"
            viewBox="0 0 640 452"
          >
            <path d="M192 184 V204" />
            <rect
              x="187.5"
              y="204.5"
              width="10"
              height="10"
              rx="2"
              className="fill-background"
            />
            <path d="M192 215 V228 C192 235 188 238 182 238 H159 C153 238 149 241 149 247 V262" />
            <path d="M192 228 C192 235 196 238 202 238 H509 C515 238 519 241 519 247 V262" />
            <path d="M337 238 C343 238 347 241 347 247 V262" />
          </svg>
          <div
            className={cn(
              "absolute top-[262px] left-11 h-[134px] w-[210px] rounded-sm border bg-background p-3",
              own
                ? "border-dashed border-control-border"
                : "border-block-border"
            )}
          >
            <div className="flex gap-x-2">
              {own ? (
                <div className="size-5 shrink-0 rounded-xs bg-control-bg-hover" />
              ) : (
                <EngineIcon engine={Engine.POSTGRES} className="size-5" />
              )}
              <div className="min-w-0">
                <div className="text-sm font-semibold text-main">
                  {own
                    ? t("workspace-setup.panel.own-instance")
                    : t("workspace-setup.panel.sample-instance")}
                </div>
                <div className={cn(CAPTION, "mt-1")}>
                  {own
                    ? t("workspace-setup.panel.connected-after-setup")
                    : t("common.instance")}
                </div>
              </div>
            </div>
            <div
              className={cn(
                "mt-3 flex items-center gap-x-1.5 truncate rounded-xs px-2.5 py-2 font-mono text-xs",
                own
                  ? "border border-dashed border-control-border text-control-light"
                  : "bg-main text-background"
              )}
            >
              <Database className="size-3.5 shrink-0" />
              {own
                ? t("workspace-setup.panel.own-databases")
                : t("workspace-setup.panel.sample-database")}
            </div>
          </div>
          <GhostInstance left={270} />
          <GhostInstance left={442} />
        </>
      )}
    </Drawn>
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
  noProject,
  teamSize,
  invitees,
}: {
  readonly screen: SetupScreen;
  readonly goal: SetupGoal | undefined;
  readonly workspaceTitle: string;
  readonly memberName: string;
  readonly memberEmail: string;
  readonly projectTitle: string;
  readonly source: SetupSource;
  // Self-hosted left the project for later.
  readonly noProject: boolean;
  readonly teamSize: GuideTeamSize | undefined;
  // The addresses Finish would add.
  readonly invitees: readonly string[];
}) {
  if (screen === "project") {
    return (
      <ProjectPreview
        workspaceTitle={workspaceTitle}
        projectTitle={projectTitle}
        source={source}
        noProject={noProject}
      />
    );
  }
  if (screen === "team") {
    return (
      <WorkspacePreview
        workspaceTitle={workspaceTitle}
        memberName={memberName}
        memberEmail={memberEmail}
        team={{ size: teamSize, invitees }}
      />
    );
  }
  // The chosen goal's picture; until a goal is chosen the workspace preview
  // stays.
  const still = screen === "goal" ? stillFor(goal) : undefined;
  if (!still) {
    return (
      <WorkspacePreview
        workspaceTitle={workspaceTitle}
        memberName={memberName}
        memberEmail={memberEmail}
      />
    );
  }
  const { width, height } = STILLS[still];
  return (
    <div
      className={cn(FRAME, "bg-background")}
      style={{ width: pictureWidth(width, height) }}
    >
      <img
        className="block h-auto w-full"
        {...STILLS[still]}
        data-still={still}
        alt=""
      />
    </div>
  );
}
