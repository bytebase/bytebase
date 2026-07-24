import type { BehaviorAnalyticsConfig, BehaviorMetric } from "./behavior";

type PostHogClient = {
  init: (apiKey: string, options?: Record<string, unknown>) => void;
  identify: (id: string, properties?: Record<string, unknown>) => void;
  opt_in_capturing?: () => void;
  opt_out_capturing?: () => void;
  reset: () => void;
  capture: (event: string, properties?: Record<string, unknown>) => void;
  startSessionRecording?: () => void;
  stopSessionRecording?: () => void;
};

type AnalyticsIdentity = {
  user: string;
  workspace: string;
};

type LoadPostHog = () => Promise<PostHogClient>;

class BehaviorAnalytics {
  private client: PostHogClient | undefined;
  private configKey: string | undefined;
  private properties: Record<string, unknown> = {};
  private enabled = false;
  private initialized = false;
  private pendingIdentity: AnalyticsIdentity | undefined;
  private pendingMetrics: BehaviorMetric[] = [];

  constructor(private readonly loadClient: LoadPostHog) {}

  async init(config: BehaviorAnalyticsConfig): Promise<void> {
    this.enabled = true;
    this.properties = config.properties ?? {};
    if (this.initialized && this.configKey === config.apiKey) {
      this.client?.opt_in_capturing?.();
      this.client?.startSessionRecording?.();
      return;
    }
    const posthog = await this.loadClient();
    if (!this.enabled) {
      return;
    }
    posthog.init(config.apiKey, config.options);
    this.client = posthog;
    this.configKey = config.apiKey;
    this.initialized = true;
    this.applyPendingIdentity();
    this.flushPendingMetrics();
  }

  identify(identity: AnalyticsIdentity): void {
    if (!this.enabled) {
      return;
    }
    this.pendingIdentity = identity;
    if (!this.client || !identity.user) {
      return;
    }
    this.applyPendingIdentity();
  }

  reset(): void {
    this.client?.reset();
    this.pendingIdentity = undefined;
    this.pendingMetrics = [];
  }

  disable(): void {
    this.enabled = false;
    this.pendingIdentity = undefined;
    this.pendingMetrics = [];
    this.client?.stopSessionRecording?.();
    this.client?.opt_out_capturing?.();
    this.client?.reset();
  }

  captureMetric(metric: BehaviorMetric): void {
    if (!this.enabled) {
      return;
    }
    if (!this.client) {
      this.pendingMetrics.push(metric);
      return;
    }
    this.captureMetricNow(metric);
  }

  private applyPendingIdentity(): void {
    if (!this.client || !this.pendingIdentity?.user) {
      return;
    }
    this.client.identify(this.pendingIdentity.user, {
      user: this.pendingIdentity.user,
      workspace: this.pendingIdentity.workspace,
    });
  }

  private flushPendingMetrics(): void {
    if (!this.client) {
      return;
    }
    for (const metric of this.pendingMetrics) {
      this.captureMetricNow(metric);
    }
    this.pendingMetrics = [];
  }

  private captureMetricNow(metric: BehaviorMetric): void {
    if (!this.client) {
      return;
    }
    const properties = { ...metric.properties, ...this.properties };
    if (this.pendingIdentity) {
      properties.user = this.pendingIdentity.user;
      properties.workspace = this.pendingIdentity.workspace;
    }
    this.client.capture(metric.event, properties);
  }
}

async function loadPostHog(): Promise<PostHogClient> {
  const mod = (await import("posthog-js")) as unknown as {
    default?: PostHogClient;
  } & PostHogClient;
  return mod.default ?? mod;
}

export function createBehaviorAnalytics(loadClient = loadPostHog) {
  return new BehaviorAnalytics(loadClient);
}

export const behaviorAnalytics = createBehaviorAnalytics();
