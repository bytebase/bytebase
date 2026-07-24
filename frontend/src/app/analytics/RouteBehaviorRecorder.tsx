import { useEffect, useMemo, useRef } from "react";
import { useCurrentRoute } from "@/app/router";
import { useAppStore } from "@/stores/app";
import type { BehaviorRecordingDecision } from "./behavior";
import {
  buildBehaviorAnalyticsConfig,
  classifyBehaviorRoute,
  createBehaviorMetric,
} from "./behavior";
import { behaviorAnalytics } from "./provider";

type ActivePageSession = {
  decision: Extract<BehaviorRecordingDecision, { recording: "allow" }>;
  startedAt: number;
  visibleStartedAt: number | undefined;
  visibleDurationMs: number;
};

const BOUNCE_THRESHOLD_MS = 5_000;
const DEFAULT_RECORDING_SAMPLE_RATE = 1;

export function RouteBehaviorRecorder() {
  const isSaaSMode = useAppStore((state) => state.isSaaSMode());
  const enableMetricCollection = useAppStore(
    (state) => state.getWorkspaceProfile().enableMetricCollection
  );
  const enabled = isSaaSMode && enableMetricCollection;

  useEffect(() => {
    if (!enabled) {
      behaviorAnalytics.disable();
    }
  }, [enabled]);

  if (!enabled) {
    return null;
  }

  return <EnabledRouteBehaviorRecorder />;
}

function EnabledRouteBehaviorRecorder() {
  const route = useCurrentRoute();
  const currentUserName = useAppStore((state) => state.currentUserName);
  const workspaceName = useAppStore((state) => state.workspaceResourceName());
  const pageSessionRef = useRef<ActivePageSession | undefined>(undefined);

  const config = useMemo(
    () =>
      buildBehaviorAnalyticsConfig({
        posthogKey: import.meta.env.BB_POSTHOG_KEY as string | undefined,
        posthogHost: import.meta.env.BB_POSTHOG_HOST as string | undefined,
        gitCommit: import.meta.env.GIT_COMMIT as string | undefined,
        recordingSampleRate: parseRecordingSampleRate(
          import.meta.env.BB_POSTHOG_RECORDING_SAMPLE_RATE as string | undefined
        ),
      }),
    []
  );

  useEffect(() => {
    if (!config) {
      return;
    }
    void behaviorAnalytics.init(config);
  }, [config]);

  useEffect(() => {
    if (!config) {
      return;
    }
    if (!currentUserName) {
      behaviorAnalytics.reset();
      return;
    }
    behaviorAnalytics.identify({
      user: currentUserName,
      workspace: workspaceName,
    });
  }, [config, currentUserName, workspaceName]);

  useEffect(() => {
    if (!config) {
      return;
    }

    const decision = classifyBehaviorRoute({
      name: route.name,
    });
    const previousSession = pageSessionRef.current;
    endPageSession(previousSession);
    if (
      previousSession &&
      decision.recording === "allow" &&
      previousSession.decision.routeId !== decision.routeId
    ) {
      behaviorAnalytics.captureMetric(
        createBehaviorMetric("page navigated", {
          properties: {
            from_route_id: previousSession.decision.routeId,
            to_route_id: decision.routeId,
          },
        })
      );
    }
    pageSessionRef.current =
      decision.recording === "allow"
        ? {
            decision,
            startedAt: Date.now(),
            visibleStartedAt: document.hidden ? undefined : Date.now(),
            visibleDurationMs: 0,
          }
        : undefined;
  }, [config, route.name, route.fullPath]);

  useEffect(() => {
    const handleVisibilityChange = () => {
      const session = pageSessionRef.current;
      if (!session) {
        return;
      }
      if (document.hidden) {
        session.visibleDurationMs += visibleSegmentMs(session);
        session.visibleStartedAt = undefined;
        return;
      }
      session.visibleStartedAt = Date.now();
    };
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      endPageSession(pageSessionRef.current);
      pageSessionRef.current = undefined;
    };
  }, []);

  return null;
}

function endPageSession(session: ActivePageSession | undefined): void {
  if (!session) {
    return;
  }
  const durationMs = Date.now() - session.startedAt;
  const visibleDurationMs =
    session.visibleDurationMs + visibleSegmentMs(session);
  behaviorAnalytics.captureMetric(
    createBehaviorMetric("page session", {
      routeId: session.decision.routeId,
      durationMs,
      visibleDurationMs,
      isBounce: durationMs < BOUNCE_THRESHOLD_MS,
    })
  );
}

function visibleSegmentMs(session: ActivePageSession): number {
  return session.visibleStartedAt === undefined
    ? 0
    : Date.now() - session.visibleStartedAt;
}

function parseRecordingSampleRate(value: string | undefined): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return DEFAULT_RECORDING_SAMPLE_RATE;
  }
  return Math.min(Math.max(parsed, 0), 1);
}
