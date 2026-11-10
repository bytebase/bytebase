import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { STANDARD_RULE_TYPES } from "@/components/sql-review/standardRules";
import { PolicyResourceType } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  devSqlReviewVersion,
  setDevSqlReviewVersion,
} from "@/utils/featureGates";

const mocks = vi.hoisted(() => ({
  canUpdate: { value: true },
  upsertPolicy: vi.fn(),
  reload: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
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

const { SQLReviewVersionSwitch } = await import("./SQLReviewVersionSwitch");

const choose = (label: string) =>
  act(async () => {
    fireEvent.click(screen.getByRole("radio", { name: label }));
  });

const click = (name: string) =>
  act(async () => {
    fireEvent.click(screen.getByRole("button", { name }));
  });

const savedRules = () => {
  const { parentPath, policy } = mocks.upsertPolicy.mock.lastCall?.[0] ?? {};
  expect(parentPath).toBe("workspaces/ws");
  expect(policy.resourceType).toBe(PolicyResourceType.WORKSPACE);
  expect(policy.enforce).toBe(true);
  return policy.policy.value.rules;
};

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

  test("switching to V1 turns every workspace standard rule off", async () => {
    render(<SQLReviewVersionSwitch />);

    await choose("V1");
    expect(
      screen.getByText("sql-review.version-switch.to-v1.description")
    ).toBeInTheDocument();
    await click("sql-review.version-switch.to-v1.confirm");

    expect(savedRules()).toEqual([]);
    expect(devSqlReviewVersion()).toBe("v1");
    expect(mocks.reload).toHaveBeenCalledTimes(1);
  });

  test("switching to V2 turns every workspace standard rule on", async () => {
    setDevSqlReviewVersion("v1");
    render(<SQLReviewVersionSwitch />);

    await choose("V2");
    await click("sql-review.version-switch.to-v2.confirm");

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
    await click("sql-review.version-switch.to-v1.confirm");

    expect(devSqlReviewVersion()).toBe("v2");
    expect(mocks.reload).not.toHaveBeenCalled();
  });

  test("is disabled without permission to update policies", () => {
    mocks.canUpdate.value = false;
    render(<SQLReviewVersionSwitch />);

    expect(screen.getByRole("radio", { name: "V1" })).toHaveAttribute(
      "aria-disabled",
      "true"
    );
  });
});
