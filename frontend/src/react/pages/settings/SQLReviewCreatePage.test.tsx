import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  fetchReviewPolicyList: vi.fn(),
}));

let SQLReviewCreatePage: typeof import("./SQLReviewCreatePage").SQLReviewCreatePage;

vi.mock("@/react/stores/sqlReview", () => ({
  useSQLReviewStore: {
    getState: () => ({
      fetchReviewPolicyList: mocks.fetchReviewPolicyList,
    }),
  },
}));

vi.mock("@/react/components/sql-review/ReviewCreation", () => ({
  ReviewCreation: () => <div data-testid="review-creation" />,
}));

beforeEach(async () => {
  mocks.fetchReviewPolicyList.mockReset();
  ({ SQLReviewCreatePage } = await import("./SQLReviewCreatePage"));
});

describe("SQLReviewCreatePage", () => {
  test("lets the review creation footer reach the bottom edge", () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    act(() => {
      root.render(<SQLReviewCreatePage />);
    });

    expect(container.firstElementChild).toHaveClass("pt-4");
    expect(container.firstElementChild).not.toHaveClass("py-4");

    act(() => {
      root.unmount();
    });
  });
});
