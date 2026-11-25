import { cleanup, fireEvent, render, waitFor } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  notify: vi.fn(),
  url: undefined as string | undefined,
  write: vi.fn(async (_text: string) => true),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: { name?: string }) =>
      options?.name ? `${key}:${options.name}` : key,
  }),
}));

// The menu's own behavior is the shared primitive's; here its parts are laid
// out in place so a test reads what the menu holds.
vi.mock("@/components/ui/dropdown-menu", () => ({
  DropdownMenu: ({ children }: { children: ReactNode }) => <>{children}</>,
  DropdownMenuContent: ({ children }: { children: ReactNode }) => (
    <div role="menu">{children}</div>
  ),
  DropdownMenuItem: ({
    children,
    onClick,
  }: {
    children: ReactNode;
    onClick?: () => void;
  }) => (
    <button onClick={onClick} role="menuitem" type="button">
      {children}
    </button>
  ),
  DropdownMenuTrigger: ({ render }: { render: ReactElement }) => render,
}));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard: (text: string) => mocks.write(text),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => ({ notify: mocks.notify }) },
}));

vi.mock("../../shell/CommentLinkContext", () => ({
  useCommentLinks: () =>
    mocks.url ? { urlOf: () => mocks.url } : undefined,
}));

import { CommentActionsMenu } from "./CommentActionsMenu";

const NAME = "projects/shop/issues/701/issueComments/c1";
const URL = "https://bytebase.example.com/projects/shop/plans/674#issueComments/c1";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.url = URL;
});
afterEach(cleanup);

const items = (container: HTMLElement) =>
  Array.from(container.querySelectorAll('[role="menuitem"]')).map(
    (item) => item.textContent
  );

describe("CommentActionsMenu", () => {
  test("is one borderless, neutral button named with the comment's author", () => {
    const { container } = render(
      <CommentActionsMenu authorName="Lee Tanaka" commentName={NAME} />
    );
    const buttons = container.querySelectorAll("button:not([role])");
    expect(buttons).toHaveLength(1);
    expect(buttons[0].getAttribute("aria-label")).toBe(
      "plan.review.thread.comment-actions:Lee Tanaka"
    );
    expect(buttons[0].className).not.toContain("border");
    expect(buttons[0].className).toContain("text-control");
  });

  test("holds Copy link alone for a reader who may not edit", () => {
    const { container } = render(
      <CommentActionsMenu authorName="Lee Tanaka" commentName={NAME} />
    );
    expect(items(container)).toEqual(["common.copy-link"]);
  });

  test("holds Edit for a reader who may edit, and starts the edit", () => {
    const onEdit = vi.fn();
    const { container, getByText } = render(
      <CommentActionsMenu
        authorName="Lee Tanaka"
        commentName={NAME}
        onEdit={onEdit}
      />
    );
    expect(items(container)).toEqual(["common.copy-link", "common.edit"]);
    fireEvent.click(getByText("common.edit"));
    expect(onEdit).toHaveBeenCalledTimes(1);
  });

  test("Copy link copies the comment's address and says so", async () => {
    const { getByText } = render(
      <CommentActionsMenu authorName="Lee Tanaka" commentName={NAME} />
    );
    fireEvent.click(getByText("common.copy-link"));
    await waitFor(() =>
      expect(mocks.notify).toHaveBeenCalledWith(
        expect.objectContaining({ style: "SUCCESS", title: "common.copied" })
      )
    );
    expect(mocks.write).toHaveBeenCalledWith(URL);
  });

  test("a failed copy says so", async () => {
    mocks.write.mockResolvedValueOnce(false);
    const { getByText } = render(
      <CommentActionsMenu authorName="Lee Tanaka" commentName={NAME} />
    );
    fireEvent.click(getByText("common.copy-link"));
    await waitFor(() =>
      expect(mocks.notify).toHaveBeenCalledWith(
        expect.objectContaining({ style: "CRITICAL", title: "common.copy-failed" })
      )
    );
  });

  test("renders nothing where a comment has no address and cannot be edited", () => {
    mocks.url = undefined;
    const { container } = render(
      <CommentActionsMenu authorName="Lee Tanaka" commentName={NAME} />
    );
    expect(container.innerHTML).toBe("");
  });
});
