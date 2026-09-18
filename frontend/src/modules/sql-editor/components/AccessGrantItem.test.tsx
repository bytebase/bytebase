import type { ReactElement } from "react";
import { act } from "react";
import { timestampFromMs } from "@bufbuild/protobuf/wkt";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  useTranslation: vi.fn(() => ({ t: (key: string) => key })),
  readStatus: vi.fn(),
  getAccessGrantDisplayStatusText: vi.fn(),
  getAccessGrantStatusTagType: vi.fn(),
}));

vi.mock("@/components/DatabaseTargetDisplay", () => ({
  DatabaseTargetDisplay: (props: {
    target: string;
    showEnvironment?: boolean;
  }) => (
    <span
      data-testid="database-target-display"
      data-target={props.target}
      data-show-environment={String(props.showEnvironment)}
    />
  ),
}));

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: mocks.useTranslation,
}));

// The real module, with the status value and its labels stubbed per test; the
// status boundary and the deadline stay real, so the clock wiring is exercised.
vi.mock("@/utils/accessGrant", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/utils/accessGrant")>();
  return {
    ...actual,
    accessGrantStatusReading: {
      read: mocks.readStatus,
      nextChangeAt: actual.accessGrantStatusReading.nextChangeAt,
    },
    getAccessGrantDisplayStatusText: mocks.getAccessGrantDisplayStatusText,
    getAccessGrantStatusTagType: mocks.getAccessGrantStatusTagType,
  };
});

// Stub Badge and Tooltip as simple pass-through
vi.mock("@/components/ui/badge", () => ({
  Badge: ({
    children,
    variant,
  }: {
    children: React.ReactNode;
    variant?: string;
  }) => (
    <span data-testid="badge" data-variant={variant}>
      {children}
    </span>
  ),
}));

vi.mock("@/components/ui/tooltip", () => ({
  // The real Tooltip renders nothing for an absent body, so the stub carries
  // the body it was given: a row that hides a reading has to offer it back.
  Tooltip: ({
    children,
    content,
  }: {
    children: React.ReactNode;
    content?: React.ReactNode;
  }) => (
    <span data-testid="tooltip" data-has-content={content !== undefined}>
      {children}
    </span>
  ),
}));

vi.mock("@/components/ui/button", () => ({
  Button: ({
    children,
    onClick,
    "data-run-btn": runBtn,
    "data-re-request-btn": reRequestBtn,
    asChild: _asChild,
    ...props
  }: {
    children: React.ReactNode;
    onClick?: (e: React.MouseEvent) => void;
    "data-run-btn"?: boolean;
    "data-re-request-btn"?: boolean;
    asChild?: boolean;
    [key: string]: unknown;
  }) => (
    <button
      data-run-btn={runBtn ? "" : undefined}
      data-re-request-btn={reRequestBtn ? "" : undefined}
      onClick={onClick}
      {...props}
    >
      {children}
    </button>
  ),
}));

vi.mock("@/lib/utils", () => ({
  cn: (...args: string[]) => args.filter(Boolean).join(" "),
}));

type AccessGrantLike = {
  name: string;
  targets: string[];
  query: string;
  unmask: boolean;
  issue: string;
  status: number;
  expiration: { case: string; value?: unknown };
};

const makeGrant = (
  overrides: Partial<AccessGrantLike> = {}
): AccessGrantLike => ({
  name: "projects/proj1/accessGrants/grant1",
  targets: ["instances/inst1/databases/db1"],
  query: "SELECT * FROM users",
  unmask: false,
  issue: "",
  status: 2, // ACTIVE
  expiration: { case: "none" },
  ...overrides,
});

let AccessGrantItem: typeof import("./AccessGrantItem").AccessGrantItem;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  document.body.appendChild(container);
  return {
    container,
    render: () => {
      act(() => {
        root.render(element);
      });
    },
    unmount: () => {
      act(() => {
        root.unmount();
      });
      container.remove();
    },
  };
};

beforeEach(async () => {
  vi.clearAllMocks();
  mocks.useTranslation.mockReturnValue({ t: (key: string) => key });
  mocks.readStatus.mockReturnValue("ACTIVE");
  mocks.getAccessGrantDisplayStatusText.mockReturnValue("Active");
  mocks.getAccessGrantStatusTagType.mockReturnValue("success");

  ({ AccessGrantItem } = await import("./AccessGrantItem"));
});

afterEach(() => {
  vi.useRealTimers();
  document.body.innerHTML = "";
});

describe("AccessGrantItem", () => {
  test("renders database targets with the unified display", () => {
    const grant = makeGrant({
      targets: [
        "instances/inst1/databases/db1",
        "instances/inst2/databases/db2",
        "instances/inst3/databases/db3",
      ],
    });

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={vi.fn()}
        onRequest={vi.fn()}
      />
    );
    render();

    const targets = Array.from(
      container.querySelectorAll("[data-testid='database-target-display']")
    );
    expect(targets).toHaveLength(2);
    expect(targets[0]?.getAttribute("data-target")).toBe(
      "instances/inst1/databases/db1"
    );
    expect(targets[1]?.getAttribute("data-target")).toBe(
      "instances/inst2/databases/db2"
    );
    expect(
      targets.every(
        (target) => target.getAttribute("data-show-environment") === "true"
      )
    ).toBe(true);
    expect(container.textContent).toContain("sql-editor.and-n-more-databases");
    unmount();
  });

  test("renders status badge with correct label for ACTIVE status", () => {
    mocks.readStatus.mockReturnValue("ACTIVE");
    mocks.getAccessGrantDisplayStatusText.mockReturnValue("Active");
    mocks.getAccessGrantStatusTagType.mockReturnValue("success");

    const grant = makeGrant();
    const onRun = vi.fn();
    const onRequest = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={onRun}
        onRequest={onRequest}
      />
    );
    render();

    const badge = container.querySelector("[data-testid='badge']");
    expect(badge).not.toBeNull();
    expect(badge!.textContent).toContain("Active");
    expect(badge!.getAttribute("data-variant")).toBe("success");
    unmount();
  });

  test("constrains an unbroken long query within the panel", () => {
    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={makeGrant({ query: "x".repeat(100_000) }) as never}
        onRun={vi.fn()}
        onRequest={vi.fn()}
      />
    );
    render();

    expect(container.firstElementChild?.className).toContain("min-w-0");
    const query = container.querySelector("p");
    expect(query?.className).toContain("w-full");
    expect(query?.className).toContain("min-w-0");
    expect(query?.className).toContain("wrap-anywhere");
    expect(query?.className).toContain("line-clamp-2");
    unmount();
  });

  test("Run button shows only for ACTIVE status", () => {
    mocks.readStatus.mockReturnValue("ACTIVE");
    const grant = makeGrant();
    const onRun = vi.fn();
    const onRequest = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={onRun}
        onRequest={onRequest}
      />
    );
    render();

    const runBtn = container.querySelector("[data-run-btn]");
    expect(runBtn).not.toBeNull();
    unmount();
  });

  test("Run button is absent for non-ACTIVE status", () => {
    mocks.readStatus.mockReturnValue("PENDING");
    const grant = makeGrant();
    const onRun = vi.fn();
    const onRequest = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={onRun}
        onRequest={onRequest}
      />
    );
    render();

    const runBtn = container.querySelector("[data-run-btn]");
    expect(runBtn).toBeNull();
    unmount();
  });

  test("Re-request button shows for REJECTED status", () => {
    mocks.readStatus.mockReturnValue("REJECTED");
    mocks.getAccessGrantDisplayStatusText.mockReturnValue("Rejected");
    mocks.getAccessGrantStatusTagType.mockReturnValue("error");

    const grant = makeGrant();
    const onRun = vi.fn();
    const onRequest = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={onRun}
        onRequest={onRequest}
      />
    );
    render();

    const reRequestBtn = container.querySelector("[data-re-request-btn]");
    expect(reRequestBtn).not.toBeNull();
    unmount();
  });

  test("Click Run → onRun(grant) called with the grant", async () => {
    mocks.readStatus.mockReturnValue("ACTIVE");
    const grant = makeGrant();
    const onRun = vi.fn();
    const onRequest = vi.fn();

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem
        grant={grant as never}
        onRun={onRun}
        onRequest={onRequest}
      />
    );
    render();

    const runBtn = container.querySelector("[data-run-btn]") as HTMLElement;
    expect(runBtn).not.toBeNull();

    await act(async () => {
      runBtn.click();
    });

    expect(onRun).toHaveBeenCalledTimes(1);
    expect(onRun).toHaveBeenCalledWith(grant);
    unmount();
  });

  test("counts down while mounted, then turns expired at the deadline", async () => {
    const actual = await vi.importActual<typeof import("@/utils/accessGrant")>(
      "@/utils/accessGrant"
    );
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-02T12:00:00Z"));
    mocks.readStatus.mockImplementation(actual.accessGrantStatusReading.read);
    mocks.useTranslation.mockReturnValue({
      t: (key: string, options?: { time?: string }) =>
        options?.time && key === "sql-editor.expire-in"
          ? `${key}:${options.time}`
          : key,
    });
    const deadlineMs = Date.now() + 3 * 60_000;
    const grant = makeGrant({
      expiration: {
        case: "expireTime",
        value: timestampFromMs(deadlineMs),
      } as never,
    });

    const { container, render, unmount } = renderIntoContainer(
      <AccessGrantItem grant={grant as never} onRun={vi.fn()} onRequest={vi.fn()} />
    );
    render();
    const advanceSeconds = (seconds: number) => {
      for (let second = 0; second < seconds; second++) {
        act(() => {
          vi.advanceTimersByTime(1_000);
        });
      }
    };

    expect(container.textContent).toContain("sql-editor.expire-in:3m");
    // A countdown is the one form that hides the deadline, so the row has to
    // offer it: a tooltip with nothing to say renders its children bare, and
    // the trigger it wraps them in is what says the offer is there.
    const countdownTooltip = Array.from(
      container.querySelectorAll("[data-testid=tooltip]")
    ).find((node) => node.textContent === "sql-editor.expire-in:3m");
    expect(countdownTooltip?.getAttribute("data-has-content")).toBe("true");

    advanceSeconds(1);
    expect(container.textContent).toContain("sql-editor.expire-in:2m");
    advanceSeconds(60);
    expect(container.textContent).toContain("sql-editor.expire-in:1m");
    advanceSeconds(60);
    expect(container.textContent).toContain("sql-editor.expire-in:0m");
    expect(container.querySelector("[data-run-btn]")).not.toBeNull();

    advanceSeconds(60);
    expect(container.textContent).toContain("issue.access-grant.expired-at");
    expect(container.querySelector("[data-run-btn]")).toBeNull();
    unmount();
  });
});
