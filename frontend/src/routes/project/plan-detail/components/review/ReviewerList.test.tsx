import { act, cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { ApprovalCandidate } from "../../../useApprovalCandidates";

const mocks = vi.hoisted(() => ({
  writeTextToClipboard: vi.fn(),
  notify: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) =>
      params && Object.keys(params).length > 0
        ? `${key}:${JSON.stringify(params)}`
        : key,
  }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

vi.mock("@/lib/clipboard", () => ({
  writeTextToClipboard: mocks.writeTextToClipboard,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => ({ notify: mocks.notify }) },
}));

vi.mock("@/components/UserAvatar", () => ({
  getAvatarColor: () => "#000",
  getInitials: (name: string) => name.slice(0, 2),
}));

// jsdom has no layout, so the real virtualizer measures a zero-height scroller
// and renders no rows; render every row instead.
vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: (options: { count: number; getItemKey?: (i: number) => string }) => ({
    getVirtualItems: () =>
      Array.from({ length: options.count }, (_, index) => ({
        index,
        key: options.getItemKey?.(index) ?? index,
        start: index * 40,
      })),
    getTotalSize: () => options.count * 40,
    measureElement: () => {},
  }),
}));

import { ReviewerList } from "./ReviewerList";

const candidate = (title: string, email: string, ineligibilities: string[] = []) =>
  ({
    canApprove: ineligibilities.length === 0,
    canReview: true,
    ineligibilities,
    user: { email, name: `users/${email}`, title },
  }) as unknown as ApprovalCandidate;

const people = (n: number) =>
  Array.from({ length: n }, (_, i) => {
    const id = String(i + 1).padStart(4, "0");
    return candidate(`Person ${id}`, `person${id}@example.com`);
  });

const rowNames = () =>
  within(screen.getByRole("list"))
    .getAllByRole("listitem")
    .map((row) => row.querySelector("[data-reviewer-name]")?.textContent);

const searchBox = () =>
  screen.queryByRole("textbox", {
    name: "plan.review.approval-flow.search-placeholder",
  });

const search = async (value: string) => {
  await act(async () => {
    fireEvent.change(searchBox()!, { target: { value } });
    vi.advanceTimersByTime(400);
  });
};

beforeEach(() => {
  vi.useFakeTimers();
  mocks.writeTextToClipboard.mockReset();
  mocks.writeTextToClipboard.mockResolvedValue(true);
  mocks.notify.mockReset();
});

afterEach(() => {
  cleanup();
  document.body.innerHTML = "";
  vi.useRealTimers();
});

describe("ReviewerList", () => {
  test("lists every candidate in order, with no cap and no summary line", () => {
    render(<ReviewerList candidates={people(120)} />);

    const names = rowNames();
    expect(names).toHaveLength(120);
    expect(names[0]).toBe("Person 0001");
    expect(names[119]).toBe("Person 0120");
    expect(document.body.textContent).not.toContain("and-n-more");
  });

  test.each([
    [10, false],
    [11, true],
  ])("with %i candidates, shows a search box: %s", (count, shown) => {
    render(<ReviewerList candidates={people(count)} />);

    expect(searchBox() !== null).toBe(shown);
  });

  test("search reaches a candidate past the 50th, by name or by email", async () => {
    render(<ReviewerList candidates={people(120)} />);

    await search("person 0103");
    expect(rowNames()).toEqual(["Person 0103"]);

    await search("PERSON0103@EXAMPLE");
    expect(rowNames()).toEqual(["Person 0103"]);
  });

  test("search keeps the list's order", async () => {
    render(
      <ReviewerList
        candidates={[
          candidate("Zoe Patel", "zoe.patel@example.com"),
          candidate("Ana Lee", "ana.lee@example.com"),
          candidate("Aaron Patel", "aaron.patel@example.com"),
          ...people(10),
        ]}
      />
    );

    await search("patel");

    expect(rowNames()).toEqual(["Zoe Patel", "Aaron Patel"]);
  });

  test("a search with no match says so, and clearing it brings the list back", async () => {
    render(<ReviewerList candidates={people(20)} />);

    await search("zz");

    expect(screen.queryByRole("list")).toBeNull();
    expect(document.body.textContent).toContain(
      'plan.review.approval-flow.no-matching-reviewers:{"query":"zz"}'
    );

    await act(async () => {
      fireEvent.click(
        screen.getByRole("button", { name: "plan.review.approval-flow.clear-search" })
      );
    });

    expect(rowNames()).toHaveLength(20);
    expect(searchBox()).toHaveValue("");
  });

  test("each row's copy button names its reviewer and copies their email", async () => {
    render(
      <ReviewerList
        candidates={[
          candidate("Ana Lee", "ana.lee@example.com"),
          candidate("deploy-bot", "deploy-bot@service.bytebase.com"),
        ]}
      />
    );

    const copyAna = screen.getByRole("button", {
      name: 'plan.review.approval-flow.copy-email:{"user":"Ana Lee"}',
    });
    screen.getByRole("button", {
      name: 'plan.review.approval-flow.copy-email:{"user":"deploy-bot"}',
    });

    await act(async () => fireEvent.click(copyAna));

    expect(mocks.writeTextToClipboard).toHaveBeenCalledExactlyOnceWith("ana.lee@example.com");
  });

  test("keeps the last plan editor in the list with the reason they can't approve", () => {
    render(
      <ReviewerList
        candidates={[candidate("Morgan Kim", "morgan.kim@example.com", ["last-plan-editor"])]}
      />
    );

    expect(rowNames()).toEqual(["Morgan Kim"]);
    expect(document.body.textContent).toContain("plan.review.candidate-last-changed-plan");
  });
});
