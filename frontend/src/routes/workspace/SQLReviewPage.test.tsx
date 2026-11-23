import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  dev: { value: true },
  sqlReviewV2: { value: true },
  fetchReviewPolicyList: vi.fn(),
  standardRules: {
    rules: undefined,
    readFailed: false,
    reload: vi.fn(),
    canEdit: true,
    setRules: vi.fn(),
    isDirty: false,
    saving: false,
    revert: vi.fn(),
    save: vi.fn(),
  },
  aiReview: {
    aiEnabled: true,
    enabled: true,
    isDirty: false,
    saving: false,
    revert: vi.fn(),
    save: vi.fn(),
  },
  aiReviewPolicy: {
    isDirty: false,
    saving: false,
    error: undefined as string | undefined,
    revert: vi.fn(),
    save: vi.fn(),
  },
  pushNotification: vi.fn(),
  useUnsavedChangesGuard: vi.fn(),
  policies: [] as {
    id: string;
    name: string;
    resources: string[];
    ruleList: unknown[];
    enforce: boolean;
  }[],
  removeReviewPolicy: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/utils/featureGates", () => ({
  sqlReviewV2Enabled: () => mocks.sqlReviewV2.value,
  devSqlReviewVersion: () => (mocks.sqlReviewV2.value ? "v2" : "v1"),
  setDevSqlReviewVersion: vi.fn(),
}));

vi.mock("./SQLReviewStandardRulesSection", async (importOriginal) => ({
  ...(await importOriginal<typeof import("./SQLReviewStandardRulesSection")>()),
  useWorkspaceStandardRules: () => mocks.standardRules,
}));

vi.mock("@/hooks/useAppState", () => ({
  useWorkspaceResourceName: () => "workspaces/ws",
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => ({ upsertPolicy: vi.fn() }) },
}));

vi.mock("./SQLReviewAIReviewSection", () => ({
  SQLReviewAIReviewSection: ({
    standardRules = true,
  }: {
    standardRules?: boolean;
  }) => (
    <div data-standard-rules={String(standardRules)}>ai-review-section</div>
  ),
  useWorkspaceAIReview: () => mocks.aiReview,
  useWorkspaceReviewAIPolicy: () => mocks.aiReviewPolicy,
}));

vi.mock("@/stores/sqlReview", () => {
  const useSQLReviewStore = () => ({
    reviewPolicyList: mocks.policies,
    removeReviewPolicy: mocks.removeReviewPolicy,
  });
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
  useUnsavedChangesGuard: mocks.useUnsavedChangesGuard,
}));
vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));
vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: () => true,
  isDev: () => mocks.dev.value,
  sqlReviewPolicySlug: () => "slug",
}));

const { SQLReviewPage } = await import("./SQLReviewPage");

const update = () =>
  act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "common.update" }));
  });

describe("SQLReviewPage", () => {
  beforeEach(() => {
    mocks.fetchReviewPolicyList.mockReset();
    for (const part of [
      mocks.standardRules,
      mocks.aiReview,
      mocks.aiReviewPolicy,
    ]) {
      part.isDirty = false;
      part.saving = false;
      part.revert.mockReset();
      part.save.mockReset().mockResolvedValue(true);
    }
    mocks.dev.value = true;
    mocks.aiReviewPolicy.error = undefined;
    mocks.aiReview.aiEnabled = true;
    mocks.aiReview.enabled = true;
    mocks.pushNotification.mockReset();
    mocks.useUnsavedChangesGuard.mockReset();
    mocks.policies = [];
    mocks.removeReviewPolicy.mockReset().mockResolvedValue(undefined);
  });

  describe("deleting a review policy", () => {
    beforeEach(() => {
      mocks.sqlReviewV2.value = false;
      mocks.policies = [
        {
          id: "reviewConfigs/prod",
          name: "Prod policy",
          resources: [],
          ruleList: [],
          enforce: true,
        },
      ];
    });

    // The table and the narrow-screen cards each have a delete button.
    const askToDelete = () =>
      fireEvent.click(
        screen.getAllByRole("button", { name: "common.delete" })[0]
      );

    test("asks in an alert dialog naming the policy, then deletes it", async () => {
      render(<SQLReviewPage />);

      askToDelete();
      const dialog = await screen.findByRole("alertdialog");
      expect(dialog).toHaveTextContent("sql-review.delete-policy-title");
      expect(dialog).toHaveTextContent("sql-review.delete-description");
      expect(mocks.removeReviewPolicy).not.toHaveBeenCalled();

      await act(async () => {
        fireEvent.click(
          within(dialog).getByRole("button", { name: "common.delete" })
        );
      });

      expect(mocks.removeReviewPolicy).toHaveBeenCalledExactlyOnceWith(
        "reviewConfigs/prod"
      );
      expect(mocks.pushNotification).toHaveBeenCalledOnce();
      await waitFor(() =>
        expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument()
      );
    });

    test("deletes nothing when the dialog is canceled", async () => {
      render(<SQLReviewPage />);

      askToDelete();
      const dialog = await screen.findByRole("alertdialog");
      fireEvent.click(
        within(dialog).getByRole("button", { name: "common.cancel" })
      );

      await waitFor(() =>
        expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument()
      );
      expect(mocks.removeReviewPolicy).not.toHaveBeenCalled();
    });
  });

  test("shows the standard rules and the AI review under SQL Review V2", () => {
    mocks.sqlReviewV2.value = true;
    render(<SQLReviewPage />);

    expect(
      screen.getByRole("heading", {
        name: /^sql-review\.standard-rules\.self/,
      })
    ).toBeInTheDocument();
    expect(screen.getByText("ai-review-section")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "common.update" })
    ).not.toBeInTheDocument();
    expect(screen.queryByText("sql-review.description")).not.toBeInTheDocument();
    expect(screen.queryByText("common.create")).not.toBeInTheDocument();
    expect(mocks.fetchReviewPolicyList).not.toHaveBeenCalled();
    expect(screen.getByText("ai-review-section")).toHaveAttribute(
      "data-standard-rules",
      "true"
    );
  });

  test("uses the normal workspace page width for SQL Review V2", () => {
    mocks.sqlReviewV2.value = true;
    const { container } = render(<SQLReviewPage />);

    expect(
      screen.getByRole("heading", {
        name: /^sql-review\.standard-rules\.self/,
      }).parentElement
    ).not.toHaveClass("mx-auto", "max-w-3xl");
    expect(
      container.querySelector('[data-slot="workspace-page-layout"]')
    ).toContainElement(
      screen.getByRole("heading", {
        name: /^sql-review\.standard-rules\.self/,
      })
    );
  });

  test("keeps the v1 review policies without SQL Review V2, with the AI review beside them", () => {
    mocks.sqlReviewV2.value = false;
    render(<SQLReviewPage />);

    expect(
      screen.queryByRole("heading", {
        name: /^sql-review\.standard-rules\.self/,
      })
    ).not.toBeInTheDocument();
    expect(screen.getByText("sql-review.policies")).toBeInTheDocument();
    expect(screen.getByText("sql-review.description")).toBeInTheDocument();
    expect(screen.getAllByText("common.create").length).toBeGreaterThan(0);
    expect(mocks.fetchReviewPolicyList).toHaveBeenCalledTimes(1);
    expect(screen.getByText("ai-review-section")).toHaveAttribute(
      "data-standard-rules",
      "false"
    );
  });

  test("uses the workspace page width for the V1 policies, AI review, and footer", () => {
    mocks.sqlReviewV2.value = false;
    mocks.aiReview.isDirty = true;
    const { container } = render(<SQLReviewPage />);
    const page = container.querySelector(
      '[data-slot="workspace-page-layout"]'
    );
    const policies = screen
      .getByRole("heading", { name: /^sql-review\.policies/ })
      .closest('[data-slot="form-section"]');
    const content = policies?.parentElement ?? null;
    const footer = container.querySelector(
      '[data-slot="sticky-action-footer-content"]'
    );

    expect(page).toContainElement(content);
    expect(content).toContainElement(screen.getByText("ai-review-section"));
    expect(content).toHaveClass("w-full");
    expect(content).not.toHaveClass("mx-auto");
    expect(content).not.toHaveClass("max-w-6xl");
    expect(footer).not.toHaveClass("mx-auto");
    expect(footer).not.toHaveClass("max-w-6xl");
  });

  test("the footer saves the AI review beside the v1 review policies", async () => {
    mocks.sqlReviewV2.value = false;
    mocks.aiReview.isDirty = true;
    render(<SQLReviewPage />);

    await update();

    expect(mocks.aiReview.save).toHaveBeenCalledTimes(1);
    expect(mocks.pushNotification).toHaveBeenCalledTimes(1);
  });

  test.each([
    true,
    false,
  ])("keeps dev-only version controls in the section heading (V2: %s)", (v2) => {
    mocks.sqlReviewV2.value = v2;
    render(<SQLReviewPage />);

    const heading = screen.getByRole("heading", {
      name: v2 ? /^sql-review\.standard-rules\.self/ : /^sql-review\.policies/,
    });
    expect(heading).toHaveAccessibleName(
      v2
        ? "sql-review.standard-rules.self sql-review.version-switch.v2"
        : "sql-review.policies sql-review.version-switch.v1"
    );
    const action = screen.getByRole("button", {
      name: v2
        ? "sql-review.version-switch.to-v1.confirm"
        : "sql-review.version-switch.to-v2.confirm",
    });
    expect(
      within(heading).getByText(
        v2 ? "sql-review.version-switch.v2" : "sql-review.version-switch.v1"
      )
    ).toBeInTheDocument();
    expect(heading).not.toContainElement(action);
    expect(
      heading.closest('[data-slot="form-section-heading-row"]')
    ).toContainElement(action);
    expect(
      screen.queryByRole("region", {
        name: "sql-review.version-switch.self",
      })
    ).not.toBeInTheDocument();
  });

  test.each([true, false])("leaves version controls out of a release build (V2: %s)", (v2) => {
    mocks.dev.value = false;
    mocks.sqlReviewV2.value = v2;
    render(<SQLReviewPage />);

    expect(screen.queryByText("sql-review.version-switch.v1")).not.toBeInTheDocument();
    expect(screen.queryByText("sql-review.version-switch.v2")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /sql-review\.version-switch\.to-v[12]\.confirm/ })).not.toBeInTheDocument();
  });

  test("the footer saves the AI review policy when only it changed", async () => {
    mocks.sqlReviewV2.value = true;
    mocks.aiReviewPolicy.isDirty = true;
    render(<SQLReviewPage />);

    await update();

    expect(mocks.aiReviewPolicy.save).toHaveBeenCalledOnce();
    expect(mocks.aiReview.save).not.toHaveBeenCalled();
    expect(mocks.standardRules.save).not.toHaveBeenCalled();
    expect(mocks.pushNotification).toHaveBeenCalledOnce();
  });

  test("does not save a hidden AI review policy draft", async () => {
    mocks.sqlReviewV2.value = true;
    mocks.aiReview.enabled = false;
    mocks.aiReview.isDirty = true;
    mocks.aiReviewPolicy.isDirty = true;
    render(<SQLReviewPage />);

    await update();

    expect(mocks.aiReview.save).toHaveBeenCalledOnce();
    expect(mocks.aiReviewPolicy.save).not.toHaveBeenCalled();
  });

  test("tracks and discards a hidden AI review policy draft", () => {
    mocks.sqlReviewV2.value = true;
    mocks.aiReview.enabled = false;
    mocks.aiReviewPolicy.isDirty = true;
    render(<SQLReviewPage />);

    expect(mocks.useUnsavedChangesGuard).toHaveBeenLastCalledWith(true);
    expect(screen.getByRole("button", { name: "common.update" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    expect(mocks.aiReviewPolicy.revert).toHaveBeenCalledOnce();
  });

  test("can save turning off AI review when its hidden policy is invalid", () => {
    mocks.sqlReviewV2.value = true;
    mocks.aiReview.enabled = false;
    mocks.aiReview.isDirty = true;
    mocks.aiReviewPolicy.isDirty = true;
    mocks.aiReviewPolicy.error = "too-large";
    render(<SQLReviewPage />);

    expect(screen.getByRole("button", { name: "common.update" })).toBeEnabled();
  });

  test("the footer cannot update while the AI review policy is invalid", () => {
    mocks.sqlReviewV2.value = true;
    mocks.aiReviewPolicy.isDirty = true;
    mocks.aiReviewPolicy.error = "too-large";
    render(<SQLReviewPage />);

    expect(screen.getByRole("button", { name: "common.update" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "common.cancel" })).toBeEnabled();
  });

  test("the footer saves only the part that changed", async () => {
    mocks.sqlReviewV2.value = true;
    mocks.aiReview.isDirty = true;
    render(<SQLReviewPage />);

    await update();

    expect(mocks.aiReview.save).toHaveBeenCalledOnce();
    expect(mocks.standardRules.save).not.toHaveBeenCalled();
    expect(mocks.pushNotification).toHaveBeenCalledOnce();
  });

  test("the footer saves every part that changed, and reports one update", async () => {
    mocks.sqlReviewV2.value = true;
    mocks.aiReview.isDirty = true;
    mocks.standardRules.isDirty = true;
    mocks.aiReviewPolicy.isDirty = true;
    render(<SQLReviewPage />);

    await update();

    expect(mocks.aiReview.save).toHaveBeenCalledOnce();
    expect(mocks.standardRules.save).toHaveBeenCalledOnce();
    expect(mocks.aiReviewPolicy.save).toHaveBeenCalledOnce();
    expect(mocks.pushNotification).toHaveBeenCalledOnce();
  });

  test.each([
    ["the AI review", mocks.aiReview],
    ["the standard rules", mocks.standardRules],
    ["the AI review policy", mocks.aiReviewPolicy],
  ])("reports no update when %s fails to save", async (_, failing) => {
    mocks.sqlReviewV2.value = true;
    mocks.aiReview.isDirty = true;
    mocks.standardRules.isDirty = true;
    mocks.aiReviewPolicy.isDirty = true;
    render(<SQLReviewPage />);
    failing.save.mockResolvedValue(false);

    await update();

    expect(mocks.aiReview.save).toHaveBeenCalledOnce();
    expect(mocks.standardRules.save).toHaveBeenCalledOnce();
    expect(mocks.aiReviewPolicy.save).toHaveBeenCalledOnce();
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("cancel reverts every part", () => {
    mocks.sqlReviewV2.value = true;
    mocks.standardRules.isDirty = true;
    render(<SQLReviewPage />);

    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));

    expect(mocks.standardRules.revert).toHaveBeenCalledOnce();
    expect(mocks.aiReview.revert).toHaveBeenCalledOnce();
    expect(mocks.aiReviewPolicy.revert).toHaveBeenCalledOnce();
  });

  test.each([
    ["the AI review", mocks.aiReview],
    ["the AI review policy", mocks.aiReviewPolicy],
  ])("the footer waits while %s is saving", (_, saving) => {
    mocks.sqlReviewV2.value = true;
    mocks.standardRules.isDirty = true;
    saving.saving = true;
    render(<SQLReviewPage />);

    expect(screen.getByRole("button", { name: "common.update" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "common.cancel" })).toBeDisabled();
  });
});
