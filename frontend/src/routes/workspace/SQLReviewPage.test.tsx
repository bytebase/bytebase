import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  sqlReviewV2: { value: true },
  fetchReviewPolicyList: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/utils/featureGates", () => ({
  sqlReviewV2Enabled: () => mocks.sqlReviewV2.value,
}));

vi.mock("./SQLReviewStandardRulesSection", () => ({
  SQLReviewStandardRulesSection: () => <div>standard-rules-section</div>,
  useWorkspaceStandardRules: () => ({
    rules: [],
    setRules: vi.fn(),
    isDirty: false,
    saving: false,
    revert: vi.fn(),
    save: vi.fn(),
  }),
}));

vi.mock("@/stores/sqlReview", () => {
  const useSQLReviewStore = () => ({ reviewPolicyList: [] });
  useSQLReviewStore.getState = () => ({
    fetchReviewPolicyList: mocks.fetchReviewPolicyList,
  });
  return { useSQLReviewStore };
});

vi.mock("@/app/router", () => ({ router: { push: vi.fn() } }));
vi.mock("@/components/sql-review/ResourceLink", () => ({
  ResourceLink: ({ resource }: { resource: string }) => <span>{resource}</span>,
}));
vi.mock("@/components/LearnMoreLink", () => ({
  LearnMoreLink: ({ children }: { children?: ReactNode }) => (
    <a href="/">{children}</a>
  ),
}));
vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: vi.fn(),
}));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: () => true,
  sqlReviewPolicySlug: () => "slug",
}));

const { SQLReviewPage } = await import("./SQLReviewPage");

describe("SQLReviewPage", () => {
  beforeEach(() => {
    mocks.fetchReviewPolicyList.mockReset();
  });

  test("shows only the standard rules under SQL Review V2", () => {
    mocks.sqlReviewV2.value = true;
    render(<SQLReviewPage />);

    expect(screen.getByText("standard-rules-section")).toBeInTheDocument();
    expect(screen.queryByText("sql-review.description")).not.toBeInTheDocument();
    expect(screen.queryByText("common.create")).not.toBeInTheDocument();
    expect(mocks.fetchReviewPolicyList).not.toHaveBeenCalled();
  });

  test("keeps the v1 review policies without SQL Review V2", () => {
    mocks.sqlReviewV2.value = false;
    render(<SQLReviewPage />);

    expect(
      screen.queryByText("standard-rules-section")
    ).not.toBeInTheDocument();
    expect(screen.getByText("sql-review.description")).toBeInTheDocument();
    expect(screen.getAllByText("common.create").length).toBeGreaterThan(0);
    expect(mocks.fetchReviewPolicyList).toHaveBeenCalledTimes(1);
  });
});
