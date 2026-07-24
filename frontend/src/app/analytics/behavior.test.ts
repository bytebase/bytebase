import { describe, expect, test } from "vitest";
import type { BehaviorMetricInput } from "./behavior";
import {
  behaviorMetricDefinitions,
  buildBehaviorAnalyticsConfig,
  classifyBehaviorRoute,
  createBehaviorMetric,
  sanitizeBehaviorProperties,
} from "./behavior";

describe("behavior analytics config", () => {
  test("stays disabled unless a PostHog key and host are both present", () => {
    expect(
      buildBehaviorAnalyticsConfig({
        posthogKey: "",
        posthogHost: "https://us.i.posthog.com",
        recordingSampleRate: 0.1,
      })
    ).toBeNull();

    expect(
      buildBehaviorAnalyticsConfig({
        posthogKey: "phc_test",
        recordingSampleRate: 0.1,
      })
    ).toBeNull();
  });

  test("uses PostHog-native pageview and recording defaults when enabled", () => {
    const config = buildBehaviorAnalyticsConfig({
      posthogKey: "phc_test",
      posthogHost: "https://us.i.posthog.com",
      recordingSampleRate: 0.25,
    });

    expect(config).toMatchObject({
      apiKey: "phc_test",
      options: {
        api_host: "https://us.i.posthog.com",
        autocapture: {
          capture_copied_text: false,
          css_selector_ignorelist: [
            ".ph-no-autocapture",
            "[data-ph-no-autocapture]",
            ".ph-no-capture",
            "[data-sensitive]",
          ],
          dom_event_allowlist: ["click", "submit"],
          element_allowlist: ["a", "button", "form", "label"],
          element_attribute_ignorelist: [
            "aria-label",
            "href",
            "title",
            "data-sensitive",
          ],
        },
        capture_pageview: "history_change",
        capture_pageleave: true,
        disable_session_recording: false,
        person_profiles: "identified_only",
        session_recording: {
          maskAllInputs: true,
          blockClass: "ph-no-capture",
          sampleRate: 0.25,
        },
      },
    });
    if (!config) {
      throw new Error("Expected behavior analytics config");
    }
    expect(
      (config.options.session_recording as Record<string, unknown>)
        .maskTextSelector
    ).toBeUndefined();
  });

  test("adds build commit metadata to PostHog event properties", () => {
    const config = buildBehaviorAnalyticsConfig({
      posthogKey: "phc_test",
      posthogHost: "https://us.i.posthog.com",
      recordingSampleRate: 0.25,
      gitCommit: "abc123",
    });

    if (!config) {
      throw new Error("Expected behavior analytics config");
    }
    expect(config.properties).toEqual({
      git_commit: "abc123",
    });

    const sanitizeProperties = config.options.sanitize_properties as (
      properties: Record<string, unknown>,
      eventName?: string
    ) => Record<string, unknown>;

    expect(
      sanitizeProperties(
        {
          $current_url: "https://cloud.bytebase.com/projects/acme?token=secret",
        },
        "$pageview"
      )
    ).toEqual({
      git_commit: "abc123",
    });
  });
});

describe("behavior analytics routes", () => {
  test("allows page sessions on any named route", () => {
    expect(
      classifyBehaviorRoute({
        name: "auth.profile.setup",
      })
    ).toMatchObject({
      routeId: "auth.profile.setup",
      recording: "allow",
    });

    expect(
      classifyBehaviorRoute({
        name: "sql-editor.database",
      })
    ).toMatchObject({
      routeId: "sql-editor.database",
      recording: "allow",
    });

    expect(
      classifyBehaviorRoute({
        name: "workspace.audit-log",
      })
    ).toMatchObject({
      routeId: "workspace.audit-log",
      recording: "allow",
    });
  });

  test("ignores missing route names", () => {
    expect(
      classifyBehaviorRoute({
        name: undefined,
      })
    ).toEqual({
      recording: "ignore",
    });
  });
});

describe("behavior analytics privacy helpers", () => {
  test("removes URL-like and raw error properties before provider capture", () => {
    expect(
      sanitizeBehaviorProperties({
        route_id: "workspace.instance.create",
        $current_url:
          "https://cloud.bytebase.com/projects/customer-a/databases?email=a@example.com",
        current_url:
          "https://cloud.bytebase.com/sql-editor/projects/p1/sheets/s1",
        raw_error: "dial tcp db.internal:5432 failed",
        safe: true,
      })
    ).toEqual({
      route_id: "workspace.instance.create",
      safe: true,
    });
  });

  test("removes page URL properties for PostHog pageviews", () => {
    expect(
      sanitizeBehaviorProperties(
        {
          $current_url:
            "https://cloud.bytebase.com/projects/customer-a/databases?email=a@example.com#token",
          $pathname: "/projects/customer-a/databases",
          current_url:
            "https://cloud.bytebase.com/sql-editor/projects/p1/sheets/s1",
        },
        "$pageview"
      )
    ).toEqual({});
  });

  test("removes visible element text from autocapture payloads", () => {
    expect(
      sanitizeBehaviorProperties(
        {
          $el_text: "users/alice@example.com",
          $elements: [
            {
              $el_text: "users/alice@example.com",
              attr__href: "/users/alice@example.com",
              tag_name: "a",
            },
          ],
          $elements_chain: "a.users attr__href=/users/alice@example.com",
          safe: true,
        },
        "$autocapture"
      )
    ).toEqual({
      safe: true,
    });
  });
});

describe("behavior analytics metrics", () => {
  test("defines metric names in one map", () => {
    expect([...behaviorMetricDefinitions.keys()]).toContain("page session");
    expect([...behaviorMetricDefinitions.keys()]).toContain("page navigated");
  });

  test("creates allowlisted page session metrics with route context", () => {
    expect(
      createBehaviorMetric("page session", {
        routeId: "workspace.instance.create",
        durationMs: 12_000,
        visibleDurationMs: 9_000,
        isBounce: false,
      })
    ).toEqual({
      event: "page session",
      properties: {
        route_id: "workspace.instance.create",
        duration_ms: 12_000,
        visible_duration_ms: 9_000,
        is_bounce: false,
      },
    });
  });

  test("creates allowlisted page navigation metrics with route transition context", () => {
    expect(
      createBehaviorMetric("page navigated", {
        properties: {
          from_route_id: "workspace.landing",
          to_route_id: "workspace.instance.create",
        },
      })
    ).toEqual({
      event: "page navigated",
      properties: {
        from_route_id: "workspace.landing",
        to_route_id: "workspace.instance.create",
      },
    });
  });

  test("keeps route transition context out of the shared metric input fields", () => {
    const routeTransitionInput = {
      // @ts-expect-error route transitions should use event-specific properties.
      fromRouteId: "workspace.landing",
    } satisfies BehaviorMetricInput;

    expect(routeTransitionInput).toEqual({
      fromRouteId: "workspace.landing",
    });
  });

  test("creates allowlisted custom action metrics and drops unsafe properties", () => {
    expect(
      createBehaviorMetric("setup guide action clicked", {
        routeId: "workspace.landing",
        resource: "projects/demo",
        properties: {
          step: "hasProject",
        },
      })
    ).toEqual({
      event: "setup guide action clicked",
      properties: {
        route_id: "workspace.landing",
        resource: "projects/demo",
        step: "hasProject",
      },
    });
  });

  test("drops unset metric properties", () => {
    const metric = createBehaviorMetric("instance connection test clicked", {
      routeId: "workspace.instance.detail",
    });

    expect(Object.keys(metric.properties)).toEqual(["route_id"]);
  });
});
