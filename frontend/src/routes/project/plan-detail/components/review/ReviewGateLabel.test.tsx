import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, test } from "vitest";
import i18n from "@/lib/i18n";
import {
  type AutomatedReviewRow,
  emptyFindings,
} from "../../utils/planGates";
import { ReviewGateLabel } from "./ReviewGateLabel";

beforeAll(async () => {
  await i18n.changeLanguage("en-US");
});

afterEach(cleanup);

const row = (overrides: Partial<AutomatedReviewRow>): AutomatedReviewRow => ({
  reviewer: "review",
  conclusion: "open",
  findings: emptyFindings(),
  holdsRollout: true,
  ...overrides,
});

describe("ReviewGateLabel", () => {
  test("draws the review's open findings as priority badges with counts outside", () => {
    const { container } = render(
      <ReviewGateLabel
        row={row({ findings: { ...emptyFindings(), open: 10, p0: 2, p1: 8 } })}
      />
    );
    expect(container.textContent).toBe("SQL review: P02P18");
    const counts = screen.getByTestId("priority-counts");
    expect(
      Array.from(counts.children, (child) =>
        Array.from(child.children, (part) => part.textContent)
      )
    ).toEqual([
      ["P0", "2"],
      ["P1", "8"],
    ]);
  });

  test("counts a plan check's errors as P0 in text", () => {
    const { container } = render(
      <ReviewGateLabel
        row={row({
          reviewer: "ghost",
          findings: { ...emptyFindings(), open: 2 },
        })}
      />
    );
    expect(container.textContent).toBe("Online migration dry run: 2 P0");
    expect(screen.queryByTestId("priority-counts")).toBeNull();
  });

  test("says why a reviewer without open findings holds the rollout", () => {
    const { container } = render(
      <ReviewGateLabel row={row({ conclusion: "not-run" })} />
    );
    expect(container.textContent).toBe("SQL review has not run");
  });
});
