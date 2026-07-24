import { describe, expect, test, vi } from "vitest";
import type { BehaviorAnalyticsConfig } from "./behavior";
import { createBehaviorAnalytics } from "./provider";

function createClient() {
  return {
    init: vi.fn(),
    identify: vi.fn(),
    reset: vi.fn(),
    capture: vi.fn(),
  };
}

const config: BehaviorAnalyticsConfig = {
  apiKey: "phc_test",
  options: {},
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
});
