import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { emptyFindings } from "../../utils/planGates";
import { PriorityCounts } from "./PriorityCounts";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

afterEach(cleanup);

describe("PriorityCounts", () => {
  test("lists the priorities with open findings, P0 first, in each one's badge color", () => {
    render(
      <PriorityCounts
        findings={{ ...emptyFindings(), open: 4, p2: 1, p0: 3 }}
      />
    );
    const counts = screen.getByTestId("priority-counts");
    expect(counts.textContent).toBe("P03P21");
    expect(screen.getByText("P0").className).toContain("text-error");
    expect(screen.getByText("P2").className).toContain("text-control");
    expect(screen.queryByText("P1")).toBeNull();
  });
});
