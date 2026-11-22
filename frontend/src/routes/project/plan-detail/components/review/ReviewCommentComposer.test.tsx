import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  issueCommentDraftKey,
  prepareIssueCommentDraftRoute,
  wasIssueCommentDraftDelivered,
} from "@/lib/issueCommentDraft";

const ISSUE = "projects/sample/issues/101";
const mocks = vi.hoisted(() => ({
  query: {} as Record<string, string>,
  createIssueComment: vi.fn(async () => undefined),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({
  useCurrentRoute: () => ({ query: mocks.query }),
}));

vi.mock("@/components/MarkdownEditor", () => ({
  MarkdownEditor: ({
    autoFocus,
    content,
    onChange,
  }: {
    autoFocus?: boolean;
    content: string;
    onChange: (value: string) => void;
  }) => (
    <textarea
      aria-label="comment-editor"
      autoFocus={autoFocus}
      onChange={(event) => onChange(event.target.value)}
      value={content}
    />
  ),
}));

vi.mock("@/components/UserAvatar", () => ({ UserAvatar: () => null }));
vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "reviewer@example.com" }),
}));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => mocks },
}));

import { ReviewCommentComposer } from "./ReviewCommentComposer";

beforeEach(() => {
  localStorage.clear();
  mocks.query = {};
  mocks.createIssueComment.mockClear();
});

afterEach(cleanup);

describe("ReviewCommentComposer", () => {
  test("opens a prefilled Agent draft without posting it", () => {
    const route = prepareIssueCommentDraftRoute(
      ISSUE,
      "Please verify the least-privilege scope."
    );
    mocks.query = route.query as Record<string, string>;

    render(<ReviewCommentComposer issueName={ISSUE} />);

    expect(screen.getByRole("textbox", { name: "comment-editor" })).toHaveProperty(
      "value",
      "Please verify the least-privilege scope."
    );
    expect(document.activeElement).toBe(
      screen.getByRole("textbox", { name: "comment-editor" })
    );
    expect(
      wasIssueCommentDraftDelivered(mocks.query.agentCommentDraft)
    ).toBe(true);
    expect(mocks.createIssueComment).not.toHaveBeenCalled();
  });

  test("shows a prepared comment while its local persistence is unavailable", async () => {
    const route = prepareIssueCommentDraftRoute(ISSUE, "Agent review");
    mocks.query = route.query as Record<string, string>;
    const key = mocks.query.agentCommentDraft;
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("Storage unavailable");
      });
    try {
      render(<ReviewCommentComposer issueName={ISSUE} />);
      expect(screen.getByRole("textbox", { name: "comment-editor" })).toHaveProperty(
        "value",
        "Agent review"
      );
      expect(sessionStorage.getItem(key)).not.toBeNull();
      expect(wasIssueCommentDraftDelivered(key)).toBe(true);
      fireEvent.click(screen.getByRole("button", { name: "common.comment" }));
      await waitFor(() => {
        expect(screen.queryByRole("textbox", { name: "comment-editor" })).toBeNull();
        expect(sessionStorage.getItem(key)).toBeNull();
      });
    } finally {
      setItem.mockRestore();
    }
  });

  test("retains an existing unsent draft on ordinary navigation", () => {
    localStorage.setItem(issueCommentDraftKey(ISSUE), "My earlier review");

    render(<ReviewCommentComposer issueName={ISSUE} />);
    expect(screen.queryByRole("textbox", { name: "comment-editor" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "plan.review.activity.add-a-comment" }));
    expect(screen.getByRole("textbox", { name: "comment-editor" })).toHaveProperty(
      "value",
      "My earlier review"
    );
  });

  test("allows ordinary review when draft storage fails, including after posting", async () => {
    const getItem = vi
      .spyOn(Storage.prototype, "getItem")
      .mockImplementation(() => {
        throw new Error("Storage unavailable");
      });
    const setItem = vi
      .spyOn(Storage.prototype, "setItem")
      .mockImplementation(() => {
        throw new Error("Storage unavailable");
      });
    const removeItem = vi
      .spyOn(Storage.prototype, "removeItem")
      .mockImplementation(() => {
        throw new Error("Storage unavailable");
      });
    try {
      render(<ReviewCommentComposer issueName={ISSUE} />);
      fireEvent.click(
        screen.getByRole("button", {
          name: "plan.review.activity.add-a-comment",
        })
      );
      fireEvent.change(screen.getByRole("textbox", { name: "comment-editor" }), {
        target: { value: "My review" },
      });
      expect(screen.getByRole("textbox", { name: "comment-editor" })).toHaveProperty(
        "value",
        "My review"
      );

      fireEvent.click(screen.getByRole("button", { name: "common.comment" }));
      await waitFor(() => {
        expect(mocks.createIssueComment).toHaveBeenCalledWith({
          issueName: ISSUE,
          comment: "My review",
        });
        expect(screen.queryByRole("textbox", { name: "comment-editor" })).toBeNull();
      });
    } finally {
      getItem.mockRestore();
      setItem.mockRestore();
      removeItem.mockRestore();
    }
  });
});
