import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { STANDARD_RULE_TYPES } from "@/components/sql-review/standardRules";
import sqlReviewEnglish from "@/locales/sql-review/en-US.json";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  devSqlReviewVersion,
  setDevSqlReviewVersion,
} from "@/utils/featureGates";

const versionDescription =
  "This badge shows the browser's selected UI version. Each project's effective standard rules determine which review version runs.";

const mocks = vi.hoisted(() => ({
  canUpdate: { value: true },
  upsertPolicy: vi.fn(),
  reload: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) =>
      (
        {
          "sql-review.version-switch.description":
            sqlReviewEnglish["version-switch"].description,
          "sql-review.version-switch.to-v1.confirm": "Switch to V1",
          "sql-review.version-switch.to-v2.confirm": "Switch to V2",
          "sql-review.version-switch.v1": "SQL Review V1",
          "sql-review.version-switch.v2": "SQL Review V2",
        } as Record<string, string>
      )[key] ?? key,
  }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useWorkspaceResourceName: () => "workspaces/ws",
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({ upsertPolicy: mocks.upsertPolicy }),
  },
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: () => mocks.canUpdate.value,
}));

const { SQLReviewVersionBadge, SQLReviewVersionSwitch } = await import(
  "./SQLReviewVersionSwitch"
);

const choose = (label: string) =>
  act(async () => {
    fireEvent.click(
      screen.getByRole("button", {
        name: `Switch to ${label}`,
      })
    );
  });

const click = (name: string) =>
  act(async () => {
    fireEvent.click(
      within(screen.getByRole("alertdialog")).getByRole("button", { name })
    );
  });

const savedRules = () => {
  const { parentPath, policy } = mocks.upsertPolicy.mock.lastCall?.[0] ?? {};
  expect(parentPath).toBe("workspaces/ws");
  expect(policy.resourceType).toBe(PolicyResourceType.WORKSPACE);
  expect(policy.enforce).toBe(true);
  return policy.policy.value.rules;
};

describe("SQLReviewVersionBadge", () => {
  afterEach(() => {
    vi.useRealTimers();
    localStorage.clear();
  });

  test.each(["v1", "v2"] as const)(
    "keeps the %s explanation in a tooltip rather than the header",
    async (version) => {
      vi.useFakeTimers();
      setDevSqlReviewVersion(version);
      render(<SQLReviewVersionBadge />);

      const badge = screen.getByText(`SQL Review ${version.toUpperCase()}`);
      expect(badge).toBeVisible();
      expect(screen.queryByText(versionDescription)).not.toBeInTheDocument();
      expect(screen.queryByRole("heading")).not.toBeInTheDocument();

      const trigger = badge.parentElement!;
      await act(async () => {
        trigger.focus();
        vi.advanceTimersByTime(100);
      });

      expect(trigger).toHaveFocus();
      const explanation = screen.getByText(versionDescription);
      expect(document.getElementById("bb-react-layer-overlay")).toContainElement(
        explanation
      );
    }
  );
});

describe("SQLReviewVersionSwitch", () => {
  const location = globalThis.location;

  beforeEach(() => {
    mocks.canUpdate.value = true;
    mocks.upsertPolicy.mockReset().mockResolvedValue(undefined);
    mocks.reload.mockReset();
    Object.defineProperty(globalThis, "location", {
      writable: true,
      value: { ...location, reload: mocks.reload },
    });
  });

  afterEach(() => {
    Object.defineProperty(globalThis, "location", {
      writable: true,
      value: location,
    });
    localStorage.clear();
  });

  test.each([
    ["v1", "Switch to V2"],
    ["v2", "Switch to V1"],
  ] as const)(
    "shows only a short switch action for %s without its own header strip",
    (version, action) => {
      setDevSqlReviewVersion(version);
      render(<SQLReviewVersionSwitch />);

      expect(
        screen.queryByRole("region", {
          name: "sql-review.version-switch.self",
        })
      ).not.toBeInTheDocument();
      expect(screen.queryByText("SQL Review V1")).not.toBeInTheDocument();
      expect(screen.queryByText("SQL Review V2")).not.toBeInTheDocument();
      expect(
        screen.queryByText("sql-review.version-switch.self")
      ).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: action })).toBeEnabled();
      expect(screen.queryByRole("radio")).not.toBeInTheDocument();
      expect(
        screen.queryByRole("heading", {
          name: "sql-review.version-switch.self",
        })
      ).not.toBeInTheDocument();
      expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    }
  );

  test("switching to V1 turns every workspace standard rule off", async () => {
    render(<SQLReviewVersionSwitch />);

    await choose("V1");
    expect(
      screen.getByText("sql-review.version-switch.to-v1.description")
    ).toBeInTheDocument();
    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    expect(devSqlReviewVersion()).toBe("v2");
    await click("Switch to V1");

    expect(savedRules()).toEqual([]);
    expect(devSqlReviewVersion()).toBe("v1");
    expect(mocks.reload).toHaveBeenCalledTimes(1);
  });

  test("switching to V2 turns every workspace standard rule on", async () => {
    setDevSqlReviewVersion("v1");
    render(<SQLReviewVersionSwitch />);

    await choose("V2");
    await click("Switch to V2");

    expect(savedRules()).toEqual([...STANDARD_RULE_TYPES]);
    expect(devSqlReviewVersion()).toBe("v2");
    expect(mocks.reload).toHaveBeenCalledTimes(1);
  });

  test("cancelling changes nothing", async () => {
    render(<SQLReviewVersionSwitch />);

    await choose("V1");
    await click("common.cancel");

    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    expect(devSqlReviewVersion()).toBe("v2");
    expect(mocks.reload).not.toHaveBeenCalled();
  });

  test("keeps this browser's version when the rules fail to save", async () => {
    mocks.upsertPolicy.mockRejectedValue(new Error("denied"));
    render(<SQLReviewVersionSwitch />);

    await choose("V1");
    await click("Switch to V1");

    expect(devSqlReviewVersion()).toBe("v2");
    expect(mocks.reload).not.toHaveBeenCalled();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();

    mocks.upsertPolicy.mockResolvedValue(undefined);
    await click("Switch to V1");

    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(2);
    expect(devSqlReviewVersion()).toBe("v1");
    expect(mocks.reload).toHaveBeenCalledOnce();
  });

  test("waits for the policy save before changing versions and prevents repeat actions", async () => {
    let resolveSave: () => void = () => {};
    mocks.upsertPolicy.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveSave = resolve;
        })
    );
    render(<SQLReviewVersionSwitch />);

    await choose("V1");
    await click("Switch to V1");

    expect(
      screen.getByRole("button", {
        name: "Switch to V1",
      })
    ).toBeDisabled();
    expect(screen.getByRole("button", { name: "common.cancel" })).toBeDisabled();
    expect(devSqlReviewVersion()).toBe("v2");
    expect(mocks.reload).not.toHaveBeenCalled();

    await act(async () => resolveSave());

    expect(devSqlReviewVersion()).toBe("v1");
    expect(mocks.reload).toHaveBeenCalledOnce();
  });

  test("is disabled without permission to update policies", () => {
    mocks.canUpdate.value = false;
    render(<SQLReviewVersionSwitch />);

    expect(
      screen.getByRole("button", { name: "Switch to V1" })
    ).toBeDisabled();
  });
});
