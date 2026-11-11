import { cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, test, vi } from "vitest";
import { InlineThreadComposer } from "./InlineThreadComposer";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: Record<string, number>) =>
      options ? `${key}:${Object.values(options).join("-")}` : key,
  }),
}));
vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: "me@example.com", title: "Me" }),
}));
vi.mock("@/components/MarkdownEditor", () => ({
  MarkdownEditor: ({ content, onChange }: { content: string; onChange: (value: string) => void }) => (
    <textarea data-testid="editor" onChange={(event) => onChange(event.target.value)} value={content} />
  ),
}));

afterEach(cleanup);

const props = {
  draft: "",
  onCancel: vi.fn(),
  onDraftChange: vi.fn(),
  onPublish: vi.fn(),
  pending: false,
  range: { startLine: 9, endLine: 10 },
};

describe("InlineThreadComposer", () => {
  test("has the card's frame, shadow included", () => {
    const { getByTestId } = render(<InlineThreadComposer {...props} />);
    const frame = getByTestId("inline-thread-composer");
    for (const name of ["border", "border-block-border", "rounded-sm", "shadow-sm", "bg-background"]) {
      expect(frame.className.split(" ")).toContain(name);
    }
  });

  test.each([
    { range: { startLine: 3, endLine: 3 }, title: "plan.review.thread.add-comment-on-line:3" },
    { range: { startLine: 9, endLine: 10 }, title: "plan.review.thread.add-comment-on-lines:9-10" },
  ])("starts with a bar titled $title, in the bar's one text color and with no fold control", ({ range, title }) => {
    const { getByTestId } = render(<InlineThreadComposer {...props} range={range} />);
    const bar = getByTestId("thread-bar");
    expect(getByTestId("inline-thread-composer").firstElementChild).toBe(bar);
    expect(bar.textContent).toBe(title);
    expect(bar.className).toContain("bg-control-bg");
    expect(bar.className).toContain("text-control");
    expect([bar, ...bar.querySelectorAll("*")].filter((node) => node.getAttribute("class")?.includes("text-control-light"))).toEqual([]);
    expect(bar.querySelector("button")).toBeNull();
  });

  test("puts the avatar in the body's column beside the editor, not in the bar", () => {
    const { getByTestId } = render(<InlineThreadComposer {...props} />);
    const avatar = getByTestId("composer-avatar");
    expect(getByTestId("thread-bar").contains(avatar)).toBe(false);
    expect(avatar.className).toContain("w-5");
    expect(avatar.nextElementSibling?.contains(getByTestId("editor"))).toBe(true);
  });

  test("publishes a non-empty draft and cancels", () => {
    const onPublish = vi.fn();
    const onCancel = vi.fn();
    const { getByText, rerender } = render(<InlineThreadComposer {...props} onCancel={onCancel} onPublish={onPublish} />);
    expect(getByText("plan.review.thread.publish").closest("button")?.disabled).toBe(true);
    rerender(<InlineThreadComposer {...props} draft="No LIMIT" onCancel={onCancel} onPublish={onPublish} />);
    fireEvent.click(getByText("plan.review.thread.publish"));
    expect(onPublish).toHaveBeenCalledExactlyOnceWith("No LIMIT");
    fireEvent.click(getByText("common.cancel"));
    expect(onCancel).toHaveBeenCalledOnce();
  });
});
