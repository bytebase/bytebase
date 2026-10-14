import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  sqlReviewV2: { value: true },
  fetchReviewPolicyList: vi.fn(),
  standardRules: {
    rules: [],
    setRules: vi.fn(),
    isDirty: false,
    saving: false,
    revert: vi.fn(),
    save: vi.fn(),
  },
  aiReview: { isDirty: false, saving: false, revert: vi.fn(), save: vi.fn() },
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/utils/featureGates", () => ({
  sqlReviewV2Enabled: () => mocks.sqlReviewV2.value,
}));

vi.mock("./SQLReviewStandardRulesSection", () => ({
  SQLReviewStandardRulesSection: () => <div>standard-rules-section</div>,
  useWorkspaceStandardRules: () => mocks.standardRules,
}));

vi.mock("./SQLReviewAIReviewSection", () => ({
  SQLReviewAIReviewSection: () => <div>ai-review-section</div>,
  useWorkspaceAIReview: () => mocks.aiReview,
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
    for (const part of [mocks.standardRules, mocks.aiReview]) {
      part.isDirty = false;
      part.saving = false;
      part.revert.mockReset();
      part.save.mockReset();
    }
  });

  test("shows the standard rules and the AI review under SQL Review V2", () => {
    mocks.sqlReviewV2.value = true;
    render(<SQLReviewPage />);

    expect(screen.getByText("standard-rules-section")).toBeInTheDocument();
    expect(screen.getByText("ai-review-section")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "common.update" })
    ).not.toBeInTheDocument();
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
    expect(screen.queryByText("ai-review-section")).not.toBeInTheDocument();
    expect(screen.getByText("sql-review.description")).toBeInTheDocument();
    expect(screen.getAllByText("common.create").length).toBeGreaterThan(0);
    expect(mocks.fetchReviewPolicyList).toHaveBeenCalledTimes(1);
  });

  test("the footer saves only the part that changed", () => {
    mocks.sqlReviewV2.value = true;
    mocks.aiReview.isDirty = true;
    render(<SQLReviewPage />);

    fireEvent.click(screen.getByRole("button", { name: "common.update" }));

    expect(mocks.aiReview.save).toHaveBeenCalledExactlyOnceWith({
      silent: false,
    });
    expect(mocks.standardRules.save).not.toHaveBeenCalled();
  });

  test("the footer saves both parts when both changed, and reports one update", () => {
    mocks.sqlReviewV2.value = true;
    mocks.aiReview.isDirty = true;
    mocks.standardRules.isDirty = true;
    render(<SQLReviewPage />);

    fireEvent.click(screen.getByRole("button", { name: "common.update" }));

    expect(mocks.aiReview.save).toHaveBeenCalledExactlyOnceWith({
      silent: true,
    });
    expect(mocks.standardRules.save).toHaveBeenCalledOnce();
  });

  test("cancel reverts both parts", () => {
    mocks.sqlReviewV2.value = true;
    mocks.standardRules.isDirty = true;
    render(<SQLReviewPage />);

    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));

    expect(mocks.standardRules.revert).toHaveBeenCalledOnce();
    expect(mocks.aiReview.revert).toHaveBeenCalledOnce();
  });

  test("the footer waits while either part is saving", () => {
    mocks.sqlReviewV2.value = true;
    mocks.standardRules.isDirty = true;
    mocks.aiReview.saving = true;
    render(<SQLReviewPage />);

    expect(screen.getByRole("button", { name: "common.update" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "common.cancel" })).toBeDisabled();
  });
});
