import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { STUB_ENVIRONMENTS } from "@/test-utils/environmentSelectStub";
import {
  DirectExecutionCallout,
  type DirectExecutionLead,
} from "./DirectExecutionCallout";
import type { DirectExecutionScope } from "@/lib/project-member/directExecution";

vi.mock("react-i18next", async () =>
  (await import("@/test-utils/i18n")).reactI18nextStub()
);

vi.mock("@/components/EnvironmentLabel", async () =>
  (await import("@/test-utils/environmentSelectStub")).environmentBadgeStub()
);

const renderCallout = (
  scope: DirectExecutionScope,
  extra: Partial<{
    lead: Exclude<DirectExecutionLead, "approver">;
    hasEnvTierFeature: boolean;
  }> = {}
) =>
  render(
    <DirectExecutionCallout
      kind="DDL/DML"
      scope={scope}
      environmentList={STUB_ENVIRONMENTS}
      hasEnvTierFeature={extra.hasEnvTierFeature ?? true}
      lead={extra.lead ?? "binding"}
    />
  );
const chips = () => screen.getAllByTestId("env-label").map((el) => el.textContent);

describe("DirectExecutionCallout", () => {
  test("some: one chip per environment, in the order given, under the surface's lead", () => {
    renderCallout(
      { type: "some", environments: ["environments/prod", "environments/staging"] },
      { lead: "grant" }
    );
    expect(chips()).toEqual(["environments/prod", "environments/staging"]);
    expect(screen.getByText(/direct-execution\.lead-grant/).textContent).toContain(
      '"kind":"DDL/DML"'
    );
  });

  test("the member drawer's lead names no one", () => {
    renderCallout({ type: "some", environments: ["environments/prod"] });
    expect(
      screen.getByText(/direct-execution\.lead-binding/).textContent
    ).toContain('"kind":"DDL/DML"');
  });

  test("some: the protected line names only the protected environments", () => {
    renderCallout({
      type: "some",
      environments: ["environments/staging", "environments/prod"],
    });
    const line = screen.getByText(/direct-execution\.protected/);
    expect(line.textContent).toContain('"count":1');
    expect(line.textContent).toContain('"environments":"Prod"');
  });

  test("some: without the environment-tier feature there is no protected line", () => {
    renderCallout(
      { type: "some", environments: ["environments/prod"] },
      { hasEnvTierFeature: false }
    );
    expect(screen.queryByText(/direct-execution\.protected/)).toBeNull();
  });

  test("some: a name the list no longer has still gets a chip", () => {
    renderCallout({ type: "some", environments: ["environments/gone"] });
    expect(chips()).toEqual(["environments/gone"]);
    expect(screen.queryByText(/direct-execution\.protected/)).toBeNull();
  });

  test("some: each lead renders its own sentence; the approver lead names the grantee", () => {
    const scope: DirectExecutionScope = {
      type: "some",
      environments: ["environments/staging"],
    };
    const { rerender } = renderCallout(scope, { lead: "request" });
    expect(screen.getByText(/lead-request/)).toBeTruthy();
    rerender(
      <DirectExecutionCallout
        kind="DML"
        lead="approver"
        grantee="Alex Kim"
        scope={scope}
        environmentList={STUB_ENVIRONMENTS}
        hasEnvTierFeature
      />
    );
    expect(screen.getByText(/lead-approver/).textContent).toContain(
      '"grantee":"Alex Kim"'
    );
  });

  test("none: one sentence for every surface, no chips", () => {
    renderCallout({ type: "none" });
    expect(screen.getByText(/direct-execution\.none/)).toBeTruthy();
    expect(screen.queryByTestId("env-label")).toBeNull();
  });

  test("all: the unscoped sentence, no chips", () => {
    renderCallout({ type: "all" });
    expect(screen.getByText(/direct-execution\.all/)).toBeTruthy();
    expect(screen.queryByTestId("env-label")).toBeNull();
  });
});
