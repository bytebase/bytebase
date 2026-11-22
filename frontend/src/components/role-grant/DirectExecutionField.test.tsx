import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { DirectExecutionField } from "./DirectExecutionField";
import type { DirectExecutionValue } from "@/lib/project-member/directExecution";

vi.mock("react-i18next", async () =>
  (await import("@/test-utils/i18n")).reactI18nextStub()
);

const featureRef = vi.hoisted(() => ({ on: true }));
vi.mock("@/hooks/useAppState", () => ({
  useEnvironmentList: () => [],
  usePlanFeature: () => featureRef.on,
}));

afterEach(() => {
  featureRef.on = true;
});

vi.mock("@/components/EnvironmentSelect", async () =>
  (await import("@/test-utils/environmentSelectStub")).environmentSelectStub()
);

vi.mock("./DirectExecutionCallout", () => ({
  DirectExecutionCallout: ({
    scope,
  }: {
    scope: { type: string; environments?: string[] };
  }) => (
    <div
      data-testid="callout"
      data-scope={scope.type}
      data-envs={scope.environments?.join(",")}
    />
  ),
}));

const renderField = (value: DirectExecutionValue, onChange = vi.fn()) => {
  render(
    <DirectExecutionField
      kind="DDL/DML"
      lead="grant"
      value={value}
      onChange={onChange}
    />
  );
  return onChange;
};
const off: DirectExecutionValue = { enabled: false, environments: [] };
const onEmpty: DirectExecutionValue = { enabled: true, environments: [] };
const onStaging: DirectExecutionValue = {
  enabled: true,
  environments: ["environments/staging"],
};

describe("DirectExecutionField", () => {
  test("off: the caption, no picker, no callout", () => {
    renderField(off);
    expect(screen.getByRole("switch").getAttribute("aria-checked")).toBe(
      "false"
    );
    expect(screen.getByText(/direct-execution\.none/).textContent).toContain(
      '"kind":"DDL/DML"'
    );
    expect(screen.queryByTestId("env-multi-select")).toBeNull();
    expect(screen.queryByTestId("callout")).toBeNull();
  });

  test("turning the switch on reports enabled and keeps the list", () => {
    const onChange = renderField({
      enabled: false,
      environments: ["environments/staging"],
    });
    fireEvent.click(screen.getByRole("switch"));
    expect(onChange).toHaveBeenCalledWith(onStaging);
  });

  test("on with nothing picked: the picker, the adjacent error, no callout", () => {
    renderField(onEmpty);
    expect(screen.getByTestId("env-multi-select")).toBeTruthy();
    expect(screen.getByText(/select-environments/)).toBeTruthy();
    expect(screen.getByText(/environment-required/)).toBeTruthy();
    expect(screen.queryByTestId("callout")).toBeNull();
    expect(screen.queryByText(/direct-execution\.none/)).toBeNull();
  });

  test("on with a pick: the callout for the picked list, no error", () => {
    renderField(onStaging);
    expect(screen.queryByText(/environment-required/)).toBeNull();
    const callout = screen.getByTestId("callout");
    expect(callout.getAttribute("data-scope")).toBe("some");
    expect(callout.getAttribute("data-envs")).toBe("environments/staging");
  });

  test("the field is controlled: picks are reported, never applied", () => {
    const onChange = renderField(onEmpty);
    fireEvent.click(screen.getByTestId("pick-staging"));
    expect(onChange).toHaveBeenCalledWith(onStaging);
    expect(screen.getByTestId("env-multi-select").getAttribute("data-value")).toBe(
      ""
    );
  });

  test("on: the picker flags the protected environment when the plan has tiers", () => {
    renderField(onStaging);
    const tags = screen.getAllByText(/direct-execution\.protected-tag/);
    expect(tags).toHaveLength(1);
    expect(
      tags[0].closest("[data-testid='env-option']")?.textContent
    ).toContain("Prod");
  });

  test("on: without environment tiers no row is flagged", () => {
    featureRef.on = false;
    renderField(onStaging);
    expect(screen.queryByText(/direct-execution\.protected-tag/)).toBeNull();
  });
});
