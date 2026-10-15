import { cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { IssueComment_ReviewMetadata_Priority } from "@/types/proto-es/v1/issue_service_pb";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: { n?: number }) =>
      options?.n !== undefined ? `${key}:${options.n}` : key,
  }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useUserByIdentifier: () => undefined,
}));

vi.mock("../PlanTargetDisplay", () => ({
  PlanTargetDisplay: ({ target }: { target: string }) => (
    <span data-testid="target">{target}</span>
  ),
}));

import { ReviewPriorityBadge, ReviewTargets } from "./ReviewResult";

afterEach(cleanup);

const targetsOf = (container: HTMLElement) =>
  Array.from(container.querySelectorAll("[data-testid='target']")).map(
    (node) => node.textContent
  );

const databases = (count: number) =>
  Array.from({ length: count }, (_, i) => `instances/i/databases/db${i}`);

describe("ReviewPriorityBadge", () => {
  test.each([
    [IssueComment_ReviewMetadata_Priority.P0, "P0", "plan.review.thread.priority.p0"],
    [IssueComment_ReviewMetadata_Priority.P1, "P1", "plan.review.thread.priority.p1"],
    [IssueComment_ReviewMetadata_Priority.P2, "P2", "plan.review.thread.priority.p2"],
  ])("labels priority %s as %s and explains it", (priority, label, description) => {
    const { getByTestId } = render(<ReviewPriorityBadge priority={priority} />);
    expect(getByTestId("review-priority").textContent).toBe(label);
    expect(getByTestId("review-priority").getAttribute("title")).toBe(description);
  });

  test("shows nothing for an unspecified priority", () => {
    const { container } = render(
      <ReviewPriorityBadge
        priority={IssueComment_ReviewMetadata_Priority.PRIORITY_UNSPECIFIED}
      />
    );
    expect(container.innerHTML).toBe("");
  });
});

describe("ReviewTargets", () => {
  test("shows a short list whole", () => {
    const { container, queryByRole } = render(
      <ReviewTargets targets={databases(4)} />
    );
    expect(targetsOf(container)).toEqual(databases(4));
    expect(queryByRole("button")).toBeNull();
  });

  test("folds a long list behind a count until expanded", () => {
    const { container, getByRole } = render(
      <ReviewTargets targets={databases(6)} />
    );
    expect(targetsOf(container)).toEqual(databases(3));
    fireEvent.click(getByRole("button", { name: "common.n-more:3" }));
    expect(targetsOf(container)).toEqual(databases(6));
  });

  test("shows nothing without targets", () => {
    const { container } = render(<ReviewTargets targets={[]} />);
    expect(container.innerHTML).toBe("");
  });
});
