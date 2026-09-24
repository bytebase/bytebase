import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { DirectExecutionCallout } from "./DirectExecutionCallout";

const { featureRef } = vi.hoisted(() => ({ featureRef: { on: true } }));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key} ${JSON.stringify(vars)}` : key,
    i18n: { language: "en-US" },
  }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useEnvironmentList: () => [
    { name: "environments/staging", title: "Staging", tags: {} },
    {
      name: "environments/prod",
      title: "Prod",
      tags: { protected: "protected" },
    },
  ],
  usePlanFeature: () => featureRef.on,
}));

vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environmentName }: { environmentName: string }) => (
    <span data-testid="env-chip">{environmentName}</span>
  ),
}));

describe("DirectExecutionCallout", () => {
  test("some: one chip per environment, in the order given", () => {
    render(
      <DirectExecutionCallout
        kind="DDL/DML"
        lead="grant"
        scope={{
          type: "some",
          environments: ["environments/prod", "environments/staging"],
        }}
      />
    );
    expect(
      screen.getAllByTestId("env-chip").map((el) => el.textContent)
    ).toEqual(["environments/prod", "environments/staging"]);
    expect(
      screen.getByText(/direct-execution\.lead-grant/).textContent
    ).toContain('"kind":"DDL/DML"');
  });

  test("some: the protected line names only protected environments, and only with the feature", () => {
    featureRef.on = true;
    const { unmount } = render(
      <DirectExecutionCallout
        kind="DDL"
        lead="binding"
        scope={{
          type: "some",
          environments: ["environments/staging", "environments/prod"],
        }}
      />
    );
    const line = screen.getByText(/direct-execution\.protected/);
    expect(line.textContent).toContain('"count":1');
    expect(line.textContent).toContain('"environments":"Prod"');
    unmount();

    featureRef.on = false;
    render(
      <DirectExecutionCallout
        kind="DDL"
        lead="binding"
        scope={{ type: "some", environments: ["environments/prod"] }}
      />
    );
    expect(screen.queryByText(/direct-execution\.protected/)).toBeNull();
    featureRef.on = true;
  });

  test("some: each lead renders its own sentence; the approver lead names the grantee", () => {
    const { rerender } = render(
      <DirectExecutionCallout
        kind="DML"
        lead="request"
        scope={{ type: "some", environments: ["environments/staging"] }}
      />
    );
    expect(screen.getByText(/lead-request/)).toBeTruthy();
    rerender(
      <DirectExecutionCallout
        kind="DML"
        lead="approver"
        grantee="Alex Kim"
        scope={{ type: "some", environments: ["environments/staging"] }}
      />
    );
    expect(screen.getByText(/lead-approver/).textContent).toContain(
      '"grantee":"Alex Kim"'
    );
    rerender(
      <DirectExecutionCallout
        kind="DML"
        lead="binding"
        scope={{ type: "some", environments: ["environments/staging"] }}
      />
    );
    expect(screen.getByText(/lead-binding/)).toBeTruthy();
  });

  test("none: the binding and grant sentences differ, and neither is a warning", () => {
    const { rerender } = render(
      <DirectExecutionCallout kind="DDL/DML" lead="binding" scope={{ type: "none" }} />
    );
    expect(screen.getByText(/none-binding/)).toBeTruthy();
    rerender(
      <DirectExecutionCallout kind="DDL/DML" lead="approver" scope={{ type: "none" }} />
    );
    expect(screen.getByText(/none-grant/)).toBeTruthy();
    expect(screen.queryByTestId("env-chip")).toBeNull();
  });

  test("all: the unscoped sentence, no chips", () => {
    render(
      <DirectExecutionCallout kind="DDL/DML" lead="binding" scope={{ type: "all" }} />
    );
    expect(screen.getByText(/direct-execution\.all/)).toBeTruthy();
    expect(screen.queryByTestId("env-chip")).toBeNull();
  });

  test("custom: the raw expression, no chip, no protected line", () => {
    const expression = 'resource.environment_id in ["prod"] || true';
    render(
      <DirectExecutionCallout
        kind="DDL/DML"
        lead="approver"
        grantee="Alex Kim"
        scope={{ type: "custom", expression }}
      />
    );
    expect(screen.getByText(expression).tagName).toBe("CODE");
    expect(screen.getByText(/direct-execution\.custom/)).toBeTruthy();
    expect(screen.queryByTestId("env-chip")).toBeNull();
    expect(screen.queryByText(/direct-execution\.protected/)).toBeNull();
  });
});
