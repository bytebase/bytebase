export type LandingStarterId =
  | "workspace-setup-first"
  | "workspace-concepts"
  | "workspace-governance"
  | "workspace-attention"
  | "workspace-recent"
  | "project-connect"
  | "project-organize"
  | "project-next"
  | "project-propose-change"
  | "project-open-blockers"
  | "project-open-recent"
  | "project-collaborate"
  | "project-access"
  | "project-review";

type Availability = "loading" | "present" | "empty" | "unknown";

export type LandingStarterInput =
  | {
      scope: "workspace";
      activeWork: Availability;
      canViewRecentActivity: boolean;
    }
  | {
      scope: "project";
      activeWork: Availability;
      database: Availability;
      change: "loading" | "not-started" | "proposed" | "rolled-out" | "unknown";
      membership: "unknown" | "solo" | "has-collaborators";
      canConnectInstance: boolean;
      canCreateChange: boolean;
      canViewRecentActivity: boolean;
    };

const workspaceSetupStarters: LandingStarterId[] = [
  "workspace-setup-first",
  "workspace-concepts",
  "workspace-governance",
];

const workspaceAttentionStarters: LandingStarterId[] = [
  "workspace-attention",
  "workspace-recent",
];

const projectGeneralStarters: LandingStarterId[] = ["project-next"];

function rankProjectStarterGroups(
  groups: LandingStarterId[][]
): LandingStarterId[] {
  return [
    ...groups.flatMap((group) => group.slice(0, 1)),
    ...groups.flatMap((group) => group.slice(1)),
  ];
}

export function getLandingStarterIds(
  input: LandingStarterInput
): LandingStarterId[] {
  if (input.scope === "workspace") {
    return input.activeWork === "present"
      ? [
          workspaceAttentionStarters[0],
          ...(input.canViewRecentActivity
            ? [workspaceAttentionStarters[1]]
            : []),
        ]
      : workspaceSetupStarters;
  }

  const groups: LandingStarterId[][] = [];

  if (input.activeWork === "present") {
    groups.push([
      "project-open-blockers",
      ...(input.canViewRecentActivity
        ? (["project-open-recent"] as const)
        : []),
    ]);
  }

  if (input.database === "present" && input.change === "not-started") {
    groups.push([
      "project-next",
      ...(input.canCreateChange ? (["project-propose-change"] as const) : []),
    ]);
  }

  if (input.database === "empty") {
    groups.push([
      ...(input.canConnectInstance ? (["project-connect"] as const) : []),
      "project-organize",
    ]);
  }

  if (input.membership === "solo") {
    groups.push(["project-collaborate", "project-access", "project-review"]);
  }

  return groups.length > 0
    ? rankProjectStarterGroups(groups)
    : projectGeneralStarters;
}
