export type BehaviorRecordingDecision =
  | {
      routeId: string;
      recording: "allow";
    }
  | {
      recording: "ignore";
    };

export type BehaviorRouteInput = {
  name?: string;
};

export type BehaviorAnalyticsConfig = {
  apiKey: string;
  options: Record<string, unknown>;
  properties?: Record<string, unknown>;
};

// PostHog recommends "[object] [verb]" event names.
// https://posthog.com/docs/product-analytics/capture-events
export type BehaviorMetricName =
  | "page session"
  | "connect database clicked"
  | "instance connection test clicked"
  | "instance create clicked"
  | "locked feature clicked"
  | "setup guide action clicked"
  | "setup guide dismissed"
  | "workspace setup completed"
  | "workspace setup skipped"
  | "post sync first change clicked"
  | "post sync sql editor clicked";

export type BehaviorMetric = {
  event: BehaviorMetricName;
  properties: Record<string, unknown>;
};

export type BehaviorMetricInput = {
  routeId?: string;
  properties?: Record<string, unknown>;
  resource?: string;
  durationMs?: number;
  visibleDurationMs?: number;
  isBounce?: boolean;
};

export function buildBehaviorAnalyticsConfig(params: {
  posthogKey?: string;
  posthogHost?: string;
  deployment: "cloud" | "self-host";
  gitCommit?: string;
  recordingSampleRate: number;
  resolveRouteId?: (url: string) => string | undefined;
}): BehaviorAnalyticsConfig | null {
  const apiKey = params.posthogKey?.trim();
  const apiHost = params.posthogHost?.trim();
  if (!apiKey || !apiHost) {
    return null;
  }
  const properties = sanitizeBehaviorProperties({
    deployment: params.deployment,
    git_commit: params.gitCommit?.trim() || undefined,
  });

  return {
    apiKey,
    properties,
    options: {
      api_host: apiHost,
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
      sanitize_properties: (
        eventProperties: Record<string, unknown>,
        eventName?: string
      ) => {
        const sanitized = sanitizeBehaviorProperties(
          eventProperties,
          eventName
        );
        const currentUrl = eventProperties.$current_url;
        const routeId =
          sanitized.route_id ??
          (typeof currentUrl === "string"
            ? params.resolveRouteId?.(currentUrl)
            : undefined);
        return {
          ...sanitized,
          ...(routeId ? { route_id: routeId } : {}),
          ...properties,
        };
      },
      session_recording: {
        maskAllInputs: true,
        blockClass: "ph-no-capture",
        sampleRate: params.recordingSampleRate,
        maskCapturedNetworkRequestFn: () => null,
      },
    },
  };
}

export function classifyBehaviorRoute(
  route: BehaviorRouteInput
): BehaviorRecordingDecision {
  const name = route.name;
  if (!name) {
    return { recording: "ignore" };
  }
  return {
    routeId: name,
    recording: "allow",
  };
}

export function sanitizeBehaviorProperties(
  properties: Record<string, unknown>,
  _eventName?: string
): Record<string, unknown> {
  const sanitized: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(properties)) {
    if (value === undefined) {
      continue;
    }
    if (isForbiddenPropertyKey(key)) {
      continue;
    }
    sanitized[key] = value;
  }
  return sanitized;
}

export function createBehaviorMetric(
  event: BehaviorMetricName,
  input: BehaviorMetricInput = {}
): BehaviorMetric {
  const customProperties = sanitizeBehaviorProperties(input.properties ?? {});
  const standardProperties = sanitizeBehaviorProperties({
    route_id: input.routeId,
    resource: input.resource,
    duration_ms: input.durationMs,
    visible_duration_ms: input.visibleDurationMs,
    is_bounce: input.isBounce,
  });

  return {
    event,
    properties: {
      ...customProperties,
      ...standardProperties,
    },
  };
}

function isForbiddenPropertyKey(key: string): boolean {
  const normalized = key.toLowerCase();
  return (
    normalized === "$el_text" ||
    normalized === "$elements" ||
    normalized === "$elements_chain" ||
    normalized === "title" ||
    normalized.includes("url") ||
    normalized.includes("path") ||
    normalized.includes("referrer") ||
    normalized.includes("host") ||
    normalized.includes("error") ||
    normalized.includes("message")
  );
}
