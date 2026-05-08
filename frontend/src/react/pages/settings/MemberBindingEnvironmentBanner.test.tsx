import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { MemberBindingEnvironmentBanner } from "./MemberBindingEnvironmentBanner";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) => {
      const parts = [key];
      if (vars) parts.push(JSON.stringify(vars));
      return parts.join(" ");
    },
  }),
}));

vi.mock("@/react/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({
    environmentName,
    className,
  }: {
    environmentName: string;
    className?: string;
  }) => (
    <span data-testid="env-label" className={className}>
      {environmentName}
    </span>
  ),
}));

describe("MemberBindingEnvironmentBanner", () => {
  test("binding-all: renders ALL environments warning, no env badges", () => {
    render(
      <MemberBindingEnvironmentBanner
        envLimitation={{ type: "unrestricted" }}
        bindingKind="DDL/DML"
      />
    );
    expect(
      screen.getByText(/project.members.ddl-current-all/)
    ).toHaveTextContent("DDL/DML");
    expect(screen.queryAllByTestId("env-label")).toHaveLength(0);
  });

  test("binding-some: renders 'in the listed environments' warning + env labels", () => {
    render(
      <MemberBindingEnvironmentBanner
        envLimitation={{
          type: "restricted",
          environments: ["environments/prod", "environments/test"],
        }}
        bindingKind="DDL"
      />
    );
    expect(
      screen.getByText(/project.members.ddl-current-some/)
    ).toHaveTextContent("DDL");
    const labels = screen.getAllByTestId("env-label");
    expect(labels).toHaveLength(2);
    expect(labels[0]).toHaveTextContent("environments/prod");
    expect(labels[1]).toHaveTextContent("environments/test");
  });

  test("binding-none: renders 'not allowed' warning, no env badges", () => {
    render(
      <MemberBindingEnvironmentBanner
        envLimitation={{ type: "restricted", environments: [] }}
        bindingKind="DML"
      />
    );
    expect(
      screen.getByText(/project.members.ddl-current-none/)
    ).toHaveTextContent("DML");
    expect(screen.queryAllByTestId("env-label")).toHaveLength(0);
  });
});
