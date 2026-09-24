import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import {
  DirectExecutionField,
  directExecutionEnvironments,
  isDirectExecutionValid,
} from "./DirectExecutionField";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, unknown>) =>
      vars ? `${key} ${JSON.stringify(vars)}` : key,
    i18n: { language: "en-US" },
  }),
}));

vi.mock("@/hooks/useAppState", () => ({
  usePlanFeature: () => true,
}));

vi.mock("@/components/EnvironmentSelect", () => ({
  EnvironmentSelect: ({
    value,
    onChange,
    placeholder,
  }: {
    value: string[];
    onChange: (next: string[]) => void;
    placeholder?: string;
  }) => (
    <div data-testid="env-select" data-value={value.join(",")}>
      <span>{placeholder}</span>
      <button
        type="button"
        data-testid="pick-staging"
        onClick={() => onChange(["environments/staging"])}
      />
      <button type="button" data-testid="clear" onClick={() => onChange([])} />
    </div>
  ),
}));

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

describe("DirectExecutionField", () => {
  test("off: the caption, no picker, no callout", () => {
    render(
      <DirectExecutionField
        kind="DDL/DML"
        lead="grant"
        value={{ enabled: false, environments: [] }}
        onChange={() => {}}
      />
    );
    expect(screen.getByRole("switch").getAttribute("aria-checked")).toBe(
      "false"
    );
    expect(screen.getByText(/off-caption/).textContent).toContain(
      '"kind":"DDL/DML"'
    );
    expect(screen.queryByTestId("env-select")).toBeNull();
    expect(screen.queryByTestId("callout")).toBeNull();
  });

  test("turning the switch on reports enabled and keeps the list", () => {
    const onChange = vi.fn();
    render(
      <DirectExecutionField
        kind="DDL"
        lead="grant"
        value={{ enabled: false, environments: ["environments/staging"] }}
        onChange={onChange}
      />
    );
    fireEvent.click(screen.getByRole("switch"));
    expect(onChange).toHaveBeenCalledWith({
      enabled: true,
      environments: ["environments/staging"],
    });
  });

  test("on with nothing picked: the picker, the adjacent error, no callout", () => {
    render(
      <DirectExecutionField
        kind="DDL/DML"
        lead="request"
        value={{ enabled: true, environments: [] }}
        onChange={() => {}}
      />
    );
    expect(screen.getByTestId("env-select")).toBeTruthy();
    expect(screen.getByText(/select-environments/)).toBeTruthy();
    expect(screen.getByText(/pick-or-off/)).toBeTruthy();
    expect(screen.queryByTestId("callout")).toBeNull();
    expect(screen.queryByText(/off-caption/)).toBeNull();
  });

  test("on with a pick: the callout for the picked list, no error", () => {
    render(
      <DirectExecutionField
        kind="DDL/DML"
        lead="request"
        value={{ enabled: true, environments: ["environments/staging"] }}
        onChange={() => {}}
      />
    );
    expect(screen.queryByText(/pick-or-off/)).toBeNull();
    const callout = screen.getByTestId("callout");
    expect(callout.getAttribute("data-scope")).toBe("some");
    expect(callout.getAttribute("data-envs")).toBe("environments/staging");
  });

  test("the field is controlled: picks are reported, never applied", () => {
    const onChange = vi.fn();
    render(
      <DirectExecutionField
        kind="DDL/DML"
        lead="grant"
        value={{ enabled: true, environments: [] }}
        onChange={onChange}
      />
    );
    fireEvent.click(screen.getByTestId("pick-staging"));
    expect(onChange).toHaveBeenCalledWith({
      enabled: true,
      environments: ["environments/staging"],
    });
    expect(screen.getByTestId("env-select").getAttribute("data-value")).toBe(
      ""
    );
  });

  test("helpers: only on-with-nothing-picked is invalid; off always submits the empty list", () => {
    expect(isDirectExecutionValid({ enabled: false, environments: [] })).toBe(
      true
    );
    expect(isDirectExecutionValid({ enabled: true, environments: [] })).toBe(
      false
    );
    expect(
      isDirectExecutionValid({
        enabled: true,
        environments: ["environments/prod"],
      })
    ).toBe(true);
    expect(
      directExecutionEnvironments({
        enabled: false,
        environments: ["environments/prod"],
      })
    ).toEqual([]);
    expect(
      directExecutionEnvironments({
        enabled: true,
        environments: ["environments/prod"],
      })
    ).toEqual(["environments/prod"]);
  });
});
