import { act, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { EnvironmentSelect } from "./EnvironmentSelect";

vi.mock("react-i18next", async () =>
  (await import("@/test-utils/i18n")).reactI18nextStub()
);

vi.mock("@/hooks/useAppState", async () => {
  const { STUB_ENVIRONMENTS } = await import(
    "@/test-utils/environmentSelectStub"
  );
  return { useEnvironmentList: () => STUB_ENVIRONMENTS };
});

vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environment }: { environment: { title: string } }) => (
    <span>{environment.title}</span>
  ),
}));

describe("EnvironmentSelect", () => {
  test("multiple: each option row is an option in a listbox and carries its suffix", () => {
    const { container } = render(
      <EnvironmentSelect
        multiple
        className="env-select"
        value={[]}
        onChange={() => {}}
        renderSuffix={(environment) =>
          environment.tags?.protected === "protected" ? (
            <span data-testid="suffix">{environment.title}</span>
          ) : null
        }
      />
    );
    act(() => {
      (container.querySelector(".env-select > div") as HTMLElement).dispatchEvent(
        new MouseEvent("click", { bubbles: true })
      );
    });
    expect(screen.getByRole("listbox")).toBeTruthy();
    expect(screen.getAllByRole("option")).toHaveLength(2);
    expect(screen.getAllByTestId("suffix").map((n) => n.textContent)).toEqual([
      "Prod",
    ]);
  });
});
