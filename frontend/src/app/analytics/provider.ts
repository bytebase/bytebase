import type { BehaviorAnalyticsConfig, BehaviorMetric } from "./behavior";

type PostHogClient = {
  init: (apiKey: string, options?: Record<string, unknown>) => void;
  identify: (id: string, properties?: Record<string, unknown>) => void;
  reset: () => void;
  capture: (event: string, properties?: Record<string, unknown>) => void;
};

type AnalyticsIdentity = {
  user: string;
  workspace: string;
};

type LoadPostHog = () => Promise<PostHogClient>;

class BehaviorAnalytics {
  private client: PostHogClient | undefined;
  private configKey: string | undefined;
  private initialized = false;
  private pendingIdentity: AnalyticsIdentity | undefined;
  private pendingMetrics: BehaviorMetric[] = [];

  constructor(private readonly loadClient: LoadPostHog) {}

  async init(config: BehaviorAnalyticsConfig): Promise<void> {
    if (this.initialized && this.configKey === config.apiKey) {
      return;
    }
    const posthog = await this.loadClient();
    posthog.init(config.apiKey, config.options);
    this.client = posthog;
    this.configKey = config.apiKey;
    this.initialized = true;
    this.applyPendingIdentity();
    this.flushPendingMetrics();
  }

  identify(identity: AnalyticsIdentity): void {
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

  captureMetric(metric: BehaviorMetric): void {
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
    const properties = { ...metric.properties };
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
