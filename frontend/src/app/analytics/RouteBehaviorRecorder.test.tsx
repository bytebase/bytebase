import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  route: {
    name: "workspace.project.database",
    fullPath: "/projects/customer-a/databases?email=a@example.com",
  },
  appStore: {
    isSaaS: true,
    enableMetricCollection: true,
    currentUserName: "users/alice@example.com",
    workspace: "workspaces/customer-a",
  },
  analytics: {
    disable: vi.fn(),
    init: vi.fn(),
    identify: vi.fn(),
    reset: vi.fn(),
    captureMetric: vi.fn(),
  },
}));

vi.mock("@/app/router", () => ({
  useCurrentRoute: () => mocks.route,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) =>
    selector({
      currentUserName: mocks.appStore.currentUserName,
      isSaaSMode: () => mocks.appStore.isSaaS,
      getWorkspaceProfile: () => ({
        enableMetricCollection: mocks.appStore.enableMetricCollection,
      }),
      workspaceResourceName: () => mocks.appStore.workspace,
    }),
}));

vi.mock("./provider", () => ({
  behaviorAnalytics: mocks.analytics,
}));

import { RouteBehaviorRecorder } from "./RouteBehaviorRecorder";

describe("RouteBehaviorRecorder", () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
    mocks.route.name = "workspace.project.database";
    mocks.route.fullPath = "/projects/customer-a/databases?email=a@example.com";
    mocks.appStore.isSaaS = true;
    mocks.appStore.enableMetricCollection = true;
    mocks.appStore.currentUserName = "users/alice@example.com";
    mocks.appStore.workspace = "workspaces/customer-a";
    vi.stubEnv("BB_POSTHOG_KEY", "phc_test");
    vi.stubEnv("BB_POSTHOG_HOST", "https://us.i.posthog.com");
    vi.stubEnv("BB_POSTHOG_RECORDING_SAMPLE_RATE", "1");
    vi.stubEnv("GIT_COMMIT", "abc123");
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.clearAllMocks();
    vi.unstubAllEnvs();
    vi.useRealTimers();
  });

  test("initializes PostHog without manually starting route recordings", async () => {
    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    expect(mocks.analytics.init).toHaveBeenCalledOnce();
    expect(mocks.analytics.identify).toHaveBeenCalledWith(
      expect.objectContaining({
        user: "users/alice@example.com",
        workspace: "workspaces/customer-a",
      })
    );
    expect(mocks.analytics.init).toHaveBeenCalledWith(
      expect.objectContaining({
        options: expect.objectContaining({
          capture_pageview: "history_change",
          disable_session_recording: false,
        }),
        properties: {
          git_commit: "abc123",
        },
      })
    );
  });

  test("does not initialize PostHog outside Bytebase Cloud", async () => {
    mocks.appStore.isSaaS = false;

    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    expect(mocks.analytics.init).not.toHaveBeenCalled();
    expect(mocks.analytics.disable).toHaveBeenCalledOnce();
  });

  test("does not initialize PostHog when workspace metric collection is disabled", async () => {
    mocks.appStore.enableMetricCollection = false;

    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    expect(mocks.analytics.init).not.toHaveBeenCalled();
    expect(mocks.analytics.disable).toHaveBeenCalledOnce();
  });

  test("disables PostHog after workspace metric collection is disabled", async () => {
    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    mocks.appStore.enableMetricCollection = false;
    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    expect(mocks.analytics.init).toHaveBeenCalledOnce();
    expect(mocks.analytics.disable).toHaveBeenCalledOnce();
  });

  test("passes replay sample rate to PostHog without local route sampling", async () => {
    vi.stubEnv("BB_POSTHOG_RECORDING_SAMPLE_RATE", "0");

    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    expect(mocks.analytics.init).toHaveBeenCalledOnce();
    expect(mocks.analytics.init).toHaveBeenCalledWith(
      expect.objectContaining({
        options: expect.objectContaining({
          session_recording: expect.objectContaining({
            sampleRate: 0,
          }),
        }),
      })
    );
  });

  test("uses default replay sample rate when the env value is invalid", async () => {
    vi.stubEnv("BB_POSTHOG_RECORDING_SAMPLE_RATE", "abc");

    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    expect(mocks.analytics.init).toHaveBeenCalledOnce();
    expect(mocks.analytics.init).toHaveBeenCalledWith(
      expect.objectContaining({
        options: expect.objectContaining({
          session_recording: expect.objectContaining({
            sampleRate: 1,
          }),
        }),
      })
    );
  });

  test("keeps PostHog initialized when entering a sensitive route", async () => {
    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    mocks.route.name = "sql-editor.database";
    mocks.route.fullPath =
      "/sql-editor/projects/customer-a/instances/prod/databases/main";

    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    expect(mocks.analytics.init).toHaveBeenCalledOnce();
  });

  test("emits a page session metric when leaving an allowed activation route", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(1_000);

    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    vi.setSystemTime(6_000);
    mocks.route.name = "workspace.instance.create";
    mocks.route.fullPath = "/instances/new";

    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    expect(mocks.analytics.captureMetric).toHaveBeenCalledWith({
      event: "page session",
      properties: expect.objectContaining({
        route_id: "workspace.project.database",
        duration_ms: 5000,
        visible_duration_ms: 5000,
      }),
    });
  });

  test("emits a route-level page navigation metric on route changes", async () => {
    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    mocks.route.name = "workspace.instance.create";
    mocks.route.fullPath = "/instances/new";

    await act(async () => {
      root.render(<RouteBehaviorRecorder />);
    });

    expect(mocks.analytics.captureMetric).toHaveBeenCalledWith({
      event: "page navigated",
      properties: {
        from_route_id: "workspace.project.database",
        to_route_id: "workspace.instance.create",
      },
    });
  });
});
