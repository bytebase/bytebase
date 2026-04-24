import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import type {
  DashboardBodyShellProps,
  DashboardShellTargets,
} from "@/react/dashboard-shell";
import { DashboardBodyShell } from "./DashboardBodyShell";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("@/react/components/header/DashboardHeader", () => ({
  DashboardHeader: ({
    showLogo,
    showMobileSidebarToggle,
    onOpenMobileSidebar,
  }: {
    showLogo: boolean;
    showMobileSidebarToggle?: boolean;
    onOpenMobileSidebar?: () => void;
  }) => (
    <button
      type="button"
      data-testid="dashboard-header"
      data-show-logo={String(showLogo)}
      data-show-mobile-toggle={String(showMobileSidebarToggle)}
      onClick={onOpenMobileSidebar}
    />
  ),
}));

function setWindowWidth(width: number) {
  act(() => {
    Object.defineProperty(window, "innerWidth", {
      configurable: true,
      writable: true,
      value: width,
    });
    window.dispatchEvent(new Event("resize"));
  });
}

function createShellHarness(
  props: Partial<DashboardBodyShellProps> = {},
  onReady = vi.fn<(targets: DashboardShellTargets) => void>()
) {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);

  const render = (nextProps: Partial<DashboardBodyShellProps> = {}) => {
    act(() => {
      root.render(
        <DashboardBodyShell
          variant="workspace"
          routeKey="/"
          onReady={onReady}
          {...props}
          {...nextProps}
        />
      );
    });
  };

  const unmount = () => {
    act(() => {
      root.unmount();
    });
    container.remove();
  };

  return {
    container,
    onReady,
    render,
    unmount,
  };
}

function getLastTargets(mock: ReturnType<typeof vi.fn>) {
  const targets = mock.mock.lastCall?.[0] as DashboardShellTargets | undefined;
  expect(targets).toBeDefined();
  if (!targets) {
    throw new Error("Expected onReady to receive shell targets");
  }
  return targets;
}

afterEach(() => {
  document.body.innerHTML = "";
  setWindowWidth(1024);
});

describe("DashboardBodyShell", () => {
  test("keeps stable shell selectors and reports targets", () => {
    setWindowWidth(1024);
    const harness = createShellHarness();
    harness.render();

    const targets = getLastTargets(harness.onReady);
    expect(targets.desktopSidebar).toBeInstanceOf(HTMLDivElement);
    expect(targets.mobileSidebar).toBeInstanceOf(HTMLDivElement);
    expect(targets.content).toBeInstanceOf(HTMLDivElement);
    expect(targets.quickstart).toBeInstanceOf(HTMLDivElement);
    expect(targets.mainContainer?.id).toBe("bb-layout-main");
    expect(
      harness.container.querySelector('[data-label="bb-main-body-wrapper"]')
    ).not.toBeNull();
    expect(
      harness.container.querySelector('[data-label="bb-dashboard-header"]')
    ).not.toBeNull();
    expect(
      harness.container.querySelector(
        '[data-testid="dashboard-header"][data-show-logo="false"][data-show-mobile-toggle="true"]'
      )
    ).not.toBeNull();
    expect(
      harness.container.querySelector(
        '[data-label="bb-dashboard-static-sidebar"]'
      )
    ).not.toBeNull();

    harness.unmount();
  });

  test("switches sidebar visibility between desktop and mobile layouts", () => {
    setWindowWidth(500);
    const harness = createShellHarness();
    harness.render();

    const targets = getLastTargets(harness.onReady);
    const desktopAside = harness.container.querySelector(
      '[data-label="bb-dashboard-static-sidebar"]'
    );
    expect(desktopAside?.className).toContain("hidden");
    expect(targets.mobileSidebar?.parentElement?.className).toContain(
      "-translate-x-full"
    );

    setWindowWidth(1024);
    harness.render();
    expect(desktopAside?.className).toContain("flex");
    expect(targets.mobileSidebar?.parentElement?.className).toContain(
      "-translate-x-full"
    );

    harness.unmount();
  });

  test("opens the mobile sidebar when the header toggle is clicked", () => {
    setWindowWidth(500);
    const harness = createShellHarness();
    harness.render();

    let targets = getLastTargets(harness.onReady);
    expect(targets.mobileSidebar?.parentElement?.className).toContain(
      "-translate-x-full"
    );

    const headerButton = harness.container.querySelector(
      '[data-testid="dashboard-header"]'
    ) as HTMLButtonElement | null;
    expect(headerButton).not.toBeNull();
    act(() => {
      headerButton?.click();
    });
    targets = getLastTargets(harness.onReady);
    expect(targets.mobileSidebar?.parentElement?.className).toContain(
      "translate-x-0"
    );

    harness.unmount();
  });

  test("closes the mobile sidebar when the route key changes", () => {
    setWindowWidth(500);
    const harness = createShellHarness();
    harness.render({ routeKey: "/a" });

    const headerButton = harness.container.querySelector(
      '[data-testid="dashboard-header"]'
    ) as HTMLButtonElement | null;
    act(() => {
      headerButton?.click();
    });

    let targets = getLastTargets(harness.onReady);
    expect(targets.mobileSidebar?.parentElement?.className).toContain(
      "translate-x-0"
    );

    harness.render({ routeKey: "/b" });
    targets = getLastTargets(harness.onReady);
    expect(targets.mobileSidebar?.parentElement?.className).toContain(
      "-translate-x-full"
    );

    harness.unmount();
  });

  test("returns null sidebar and header targets for workspace root", () => {
    const harness = createShellHarness({ isRootPath: true });
    harness.render();

    const targets = getLastTargets(harness.onReady);
    expect(targets.desktopSidebar).toBeNull();
    expect(targets.mobileSidebar).toBeNull();
    expect(targets.content).toBeInstanceOf(HTMLDivElement);
    expect(
      harness.container.querySelector('[data-testid="dashboard-header"]')
    ).toBeNull();

    harness.unmount();
  });

  test("uses the issues variant without sidebar targets", () => {
    const container = document.createElement("div");
    document.body.appendChild(container);
    const root: Root = createRoot(container);
    const onReady = vi.fn<(targets: DashboardShellTargets) => void>();

    act(() => {
      root.render(
        <DashboardBodyShell
          variant="issues"
          routeKey="/issues"
          onReady={onReady}
        />
      );
    });

    const targets = getLastTargets(onReady);
    expect(targets.desktopSidebar).toBeNull();
    expect(targets.mobileSidebar).toBeNull();
    expect(
      container.querySelector('[data-label="bb-main-body-wrapper"]')?.className
    ).toContain("border-x");
    expect(
      container.querySelector(
        '[data-testid="dashboard-header"][data-show-logo="true"][data-show-mobile-toggle="false"]'
      )
    ).not.toBeNull();

    act(() => {
      root.unmount();
    });
    container.remove();
  });
});
