import { describe, expect, test } from "vitest";
import {
  getLandingStarterIdForText,
  getLandingStarterIds,
} from "./landingStarters";

describe("getLandingStarterIdForText", () => {
  test("does not special-case a typed project readiness starter", () => {
    expect(
      getLandingStarterIdForText({
        context: "projects/orders",
        text: "What can I do next?",
        translate: (key) =>
          key === "agent.landing-starters.project-next"
            ? "What can I do next?"
            : key,
      })
    ).toBeUndefined();
  });

  test("does not infer a project starter in workspace context", () => {
    expect(
      getLandingStarterIdForText({
        context: "workspaces/demo",
        text: "What can I do next?",
        translate: (key) =>
          key === "agent.landing-starters.project-next"
            ? "What can I do next?"
            : key,
      })
    ).toBeUndefined();
  });
});

describe("getLandingStarterIds", () => {
  test.each([
    {
      activeWork: "present",
      database: "empty",
      change: "not-started",
      membership: "solo",
    },
    {
      activeWork: "empty",
      database: "present",
      change: "rolled-out",
      membership: "has-collaborators",
    },
    {
      activeWork: "unknown",
      database: "unknown",
      change: "unknown",
      membership: "unknown",
    },
  ] as const)("always offers project-next for project state %#", (state) => {
    expect(
      getLandingStarterIds({
        scope: "project",
        ...state,
        canConnectInstance: true,
        canCreateChange: true,
        canViewRecentActivity: true,
      })
    ).toContain("project-next");
  });

  test("offers setup guidance in a workspace without setup resources", () => {
    expect(
      getLandingStarterIds({
        scope: "workspace",
        setupResources: "empty",
        canViewRecentActivity: false,
      })
    ).toEqual([
      "workspace-setup-first",
      "workspace-concepts",
      "workspace-governance",
    ]);
  });

  test("offers normal workspace prompts when setup resources exist", () => {
    expect(
      getLandingStarterIds({
        scope: "workspace",
        setupResources: "present",
        canViewRecentActivity: true,
      })
    ).toEqual(["workspace-attention", "workspace-recent"]);
  });

  test.each(["loading", "unknown"] as const)(
    "does not infer missing setup resources while discovery is %s",
    (setupResources) => {
      expect(
        getLandingStarterIds({
          scope: "workspace",
          setupResources,
          canViewRecentActivity: false,
        })
      ).toEqual(["workspace-attention"]);
    }
  );

  test("does not offer workspace recent activity without audit-log access", () => {
    expect(
      getLandingStarterIds({
        scope: "workspace",
        setupResources: "present",
        canViewRecentActivity: false,
      })
    ).toEqual(["workspace-attention"]);
  });

  test("combines open work, database setup, and solo collaboration prompts", () => {
    expect(
      getLandingStarterIds({
        scope: "project",
        activeWork: "present",
        database: "empty",
        change: "not-started",
        membership: "solo",
        canConnectInstance: true,
        canCreateChange: true,
        canViewRecentActivity: true,
      })
    ).toEqual([
      "project-open-blockers",
      "project-connect",
      "project-collaborate",
      "project-open-recent",
      "project-organize",
      "project-access",
      "project-review",
      "project-next",
    ]);
  });

  test.each(["loading", "unknown"] as const)(
    "combines known database and collaboration prompts while active work discovery is %s",
    (activeWork) => {
      expect(
        getLandingStarterIds({
          scope: "project",
          activeWork,
          database: "present",
          change: "not-started",
          membership: "solo",
          canConnectInstance: true,
          canCreateChange: true,
          canViewRecentActivity: true,
        })
      ).toEqual([
        "project-next",
        "project-collaborate",
        "project-propose-change",
        "project-access",
        "project-review",
      ]);
    }
  );

  test("offers project discovery after database access is confirmed", () => {
    expect(
      getLandingStarterIds({
        scope: "project",
        activeWork: "empty",
        database: "present",
        change: "not-started",
        membership: "has-collaborators",
        canConnectInstance: true,
        canCreateChange: true,
        canViewRecentActivity: true,
      })
    ).toEqual(["project-next", "project-propose-change"]);
  });

  test("does not offer a database-change prompt to a member who cannot create one", () => {
    expect(
      getLandingStarterIds({
        scope: "project",
        activeWork: "empty",
        database: "present",
        change: "not-started",
        membership: "has-collaborators",
        canConnectInstance: true,
        canCreateChange: false,
        canViewRecentActivity: true,
      })
    ).toEqual(["project-next"]);
  });

  test("does not offer first-change prompts after a database change is proposed", () => {
    expect(
      getLandingStarterIds({
        scope: "project",
        activeWork: "present",
        database: "present",
        change: "proposed",
        membership: "has-collaborators",
        canConnectInstance: true,
        canCreateChange: true,
        canViewRecentActivity: true,
      })
    ).toEqual(["project-open-blockers", "project-open-recent", "project-next"]);
  });

  test("offers connection guidance only when the member can connect an instance", () => {
    expect(
      getLandingStarterIds({
        scope: "project",
        activeWork: "empty",
        database: "empty",
        change: "not-started",
        membership: "has-collaborators",
        canConnectInstance: false,
        canCreateChange: false,
        canViewRecentActivity: true,
      })
    ).toEqual(["project-organize", "project-next"]);
  });

  test("offers collaboration prompts to a solo project member", () => {
    expect(
      getLandingStarterIds({
        scope: "project",
        activeWork: "empty",
        database: "present",
        change: "rolled-out",
        membership: "solo",
        canConnectInstance: true,
        canCreateChange: true,
        canViewRecentActivity: true,
      })
    ).toEqual([
      "project-collaborate",
      "project-access",
      "project-review",
      "project-next",
    ]);
  });

  test.each(["loading", "unknown"] as const)(
    "falls back to answerable project guidance when database discovery is %s",
    (database) => {
      expect(
        getLandingStarterIds({
          scope: "project",
          activeWork: "unknown",
          database,
          change: "unknown",
          membership: "unknown",
          canConnectInstance: true,
          canCreateChange: false,
          canViewRecentActivity: true,
        })
      ).toEqual(["project-next"]);
    }
  );

  test("does not offer recent activity without audit-log access", () => {
    expect(
      getLandingStarterIds({
        scope: "project",
        activeWork: "present",
        database: "present",
        change: "rolled-out",
        membership: "has-collaborators",
        canConnectInstance: true,
        canCreateChange: false,
        canViewRecentActivity: false,
      })
    ).toEqual(["project-open-blockers", "project-next"]);
  });
});
