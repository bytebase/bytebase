import { describe, expect, test, vi } from "vitest";
import type { BehaviorAnalyticsConfig } from "./behavior";
import { createBehaviorAnalytics } from "./provider";

function createClient() {
  return {
    init: vi.fn(),
    identify: vi.fn(),
    opt_in_capturing: vi.fn(),
    opt_out_capturing: vi.fn(),
    reset: vi.fn(),
    capture: vi.fn(),
    startSessionRecording: vi.fn(),
    stopSessionRecording: vi.fn(),
  };
}

const config: BehaviorAnalyticsConfig = {
  apiKey: "phc_test",
  options: {},
};

const configWithProperties: BehaviorAnalyticsConfig = {
  apiKey: "phc_test",
  options: {},
  properties: {
    git_commit: "abc123",
  },
};

describe("BehaviorAnalytics provider", () => {
  test("applies pending identify requests after async init", async () => {
    const client = createClient();
    const analytics = createBehaviorAnalytics(
      () => new Promise((resolve) => setTimeout(() => resolve(client), 0))
    );

    const init = analytics.init(config);
    analytics.identify({
      user: "users/alice@example.com",
      workspace: "workspaces/acme",
    });

    await init;

    expect(client.identify).toHaveBeenCalledWith("users/alice@example.com", {
      user: "users/alice@example.com",
      workspace: "workspaces/acme",
    });
  });

  test("queues metrics captured before async init completes", async () => {
    const client = createClient();
    const analytics = createBehaviorAnalytics(
      () => new Promise((resolve) => setTimeout(() => resolve(client), 0))
    );

    const init = analytics.init(config);
    analytics.identify({
      user: "users/alice@example.com",
      workspace: "workspaces/acme",
    });
    analytics.captureMetric({
      event: "connect database clicked",
      properties: {
        route_id: "workspace.project.database",
      },
    });

    await init;

    expect(client.capture).toHaveBeenCalledWith("connect database clicked", {
      route_id: "workspace.project.database",
      user: "users/alice@example.com",
      workspace: "workspaces/acme",
    });
  });

  test("captures metrics without empty identity properties before identify", async () => {
    const client = createClient();
    const analytics = createBehaviorAnalytics(() => Promise.resolve(client));

    await analytics.init(config);
    analytics.captureMetric({
      event: "connect database clicked",
      properties: {
        route_id: "workspace.project.database",
      },
    });

    expect(client.capture.mock.calls[0]?.[1]).toStrictEqual({
      route_id: "workspace.project.database",
    });
  });

  test("adds global config properties to captured metrics", async () => {
    const client = createClient();
    const analytics = createBehaviorAnalytics(() => Promise.resolve(client));

    await analytics.init(configWithProperties);
    analytics.captureMetric({
      event: "connect database clicked",
      properties: {
        route_id: "workspace.project.database",
      },
    });

    expect(client.capture).toHaveBeenCalledWith("connect database clicked", {
      git_commit: "abc123",
      route_id: "workspace.project.database",
    });
  });

  test("opts out and ignores identity and metrics while disabled", async () => {
    const client = createClient();
    const analytics = createBehaviorAnalytics(() => Promise.resolve(client));

    await analytics.init(config);
    analytics.disable();
    analytics.identify({
      user: "users/alice@example.com",
      workspace: "workspaces/acme",
    });
    analytics.captureMetric({
      event: "connect database clicked",
      properties: {
        route_id: "workspace.project.database",
      },
    });

    expect(client.stopSessionRecording).toHaveBeenCalledOnce();
    expect(client.opt_out_capturing).toHaveBeenCalledOnce();
    expect(client.reset).toHaveBeenCalledOnce();
    expect(client.identify).not.toHaveBeenCalled();
    expect(client.capture).not.toHaveBeenCalled();
  });

  test("drops pending metrics when disabled before async init completes", async () => {
    const client = createClient();
    const analytics = createBehaviorAnalytics(
      () => new Promise((resolve) => setTimeout(() => resolve(client), 0))
    );

    const init = analytics.init(config);
    analytics.captureMetric({
      event: "connect database clicked",
      properties: {
        route_id: "workspace.project.database",
      },
    });
    analytics.disable();

    await init;

    expect(client.init).not.toHaveBeenCalled();
    expect(client.capture).not.toHaveBeenCalled();
  });

  test("restarts session recording when re-enabled after disable", async () => {
    const client = createClient();
    const analytics = createBehaviorAnalytics(() => Promise.resolve(client));

    await analytics.init(config);
    analytics.disable();
    await analytics.init(config);

    expect(client.opt_in_capturing).toHaveBeenCalledOnce();
    expect(client.startSessionRecording).toHaveBeenCalledOnce();
  });
});
