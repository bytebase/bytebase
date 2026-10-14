import { ArrowDown } from "lucide-react";
import { useTranslation } from "react-i18next";
import accessControlsImage from "@/assets/illustration/setup-access-controls.webp";
import dataMaskingImage from "@/assets/illustration/setup-data-masking.webp";
import heroImage from "@/assets/illustration/setup-hero.webp";
import { cn } from "@/lib/utils";
import type { SetupGoal, SetupScreen, SetupSource } from "./draft";

// Each still with its pixel size, which holds the picture's place in the
// panel until the file has loaded.
const STILLS = {
  hero: { src: heroImage, width: 1222, height: 829 },
  "data-masking": { src: dataMaskingImage, width: 1368, height: 1192 },
  "access-controls": { src: accessControlsImage, width: 1368, height: 1192 },
} as const;

type Still = keyof typeof STILLS;

// One picture per screen: what Bytebase does, then the chosen goal, then who
// gets access. The hero is a change on its way through review, so it is also
// the change goal's picture; "just exploring" commits to no goal and keeps it.
const stillFor = (
  screen: Exclude<SetupScreen, "project">,
  goal: SetupGoal | undefined
): Still => {
  if (screen === "team") return "access-controls";
  if (screen === "goal" && goal === "query-data") return "data-masking";
  return "hero";
};

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
  projectTitle,
  source,
  isSaaS,
}: {
  readonly screen: SetupScreen;
  readonly goal: SetupGoal | undefined;
  readonly projectTitle: string;
  readonly source: SetupSource;
  readonly isSaaS: boolean;
}) {
  const { t } = useTranslation();

  if (screen !== "project") {
    const still = stillFor(screen, goal);
    return (
      <div className="w-full max-w-lg overflow-hidden rounded-sm border border-block-border bg-background">
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
