import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { DDLWarningCallout } from "./DDLWarningCallout";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) => {
      const parts = [key];
      if (vars) parts.push(JSON.stringify(vars));
      return parts.join(" ");
    },
  }),
}));

describe("DDLWarningCallout", () => {
  test("drawer variant renders ddl-warning copy with kind interpolated", () => {
    render(<DDLWarningCallout type="drawer" kind="DDL/DML" />);
    expect(screen.getByText(/project.members.ddl-warning/)).toHaveTextContent(
      "DDL/DML"
    );
  });

  test("binding-some renders ddl-current-some copy", () => {
    render(<DDLWarningCallout type="binding-some" kind="DML" />);
    expect(
      screen.getByText(/project.members.ddl-current-some/)
    ).toHaveTextContent("DML");
    expect(screen.queryByText(/Staging/)).not.toBeInTheDocument();
  });

  test("binding-all renders ddl-current-all copy", () => {
    render(<DDLWarningCallout type="binding-all" kind="DDL/DML" />);
    expect(
      screen.getByText(/project.members.ddl-current-all/)
    ).toHaveTextContent("DDL/DML");
  });

  test("binding-none renders ddl-current-none copy", () => {
    render(<DDLWarningCallout type="binding-none" kind="DML" />);
    expect(
      screen.getByText(/project.members.ddl-current-none/)
    ).toHaveTextContent("DML");
  });

  test("renders an alert with role=alert", () => {
    render(<DDLWarningCallout type="drawer" kind="DDL" />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });
});

const _typeChecks = () => {
  // biome-ignore format: keep @ts-expect-error directive aligned with JSX line
  // @ts-expect-error — `binding-all` does not accept extra props.
  const _a = <DDLWarningCallout type="binding-all" kind="DDL" environments={["x"]} />;
  // biome-ignore format: keep @ts-expect-error directive aligned with JSX line
  // @ts-expect-error — `drawer` does not accept extra props.
  const _b = <DDLWarningCallout type="drawer" kind="DDL" environments={["x"]} />;
  return [_a, _b];
};
void _typeChecks;
