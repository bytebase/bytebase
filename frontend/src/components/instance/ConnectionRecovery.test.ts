import { readFileSync } from "node:fs";
import { join } from "node:path";
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  ConnectionRecovery,
  normalizeConnectionFailureCategory,
} from "./ConnectionRecovery";

const mocks = vi.hoisted(() => ({ isSaaSMode: false }));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: { isSaaSMode: () => boolean }) => unknown) =>
    selector({ isSaaSMode: () => mocks.isSaaSMode }),
}));

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("ConnectionRecovery", () => {
  beforeEach(() => {
    mocks.isSaaSMode = false;
  });

  test("normalizes backend connection failure categories", () => {
    expect(normalizeConnectionFailureCategory("auth_failed")).toBe(
      "auth_failed"
    );
    expect(normalizeConnectionFailureCategory("ssl_tls_failed")).toBe(
      "ssl_tls_failed"
    );
    expect(normalizeConnectionFailureCategory("not_a_category")).toBe(
      "unknown"
    );
    expect(normalizeConnectionFailureCategory(undefined)).toBe("unknown");
  });

  test("renders recovery with the shared warning alert", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/ConnectionRecovery.tsx"),
      "utf-8"
    );

    expect(source).toContain('from "@/components/ui/alert"');
    expect(source).toContain("<Alert");
    expect(source).toContain('variant="warning"');
    expect(source).not.toContain("border-control-border bg-control-bg");
    expect(source).not.toContain("toLowerCase");
  });

  test("shows category-specific recovery steps", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/ConnectionRecovery.tsx"),
      "utf-8"
    );

    expect(source).toContain("steps:");
    expect(source).toContain("recovery.steps.map");
    expect(source).toContain("<ul");
    expect(source).toContain("<li");
    expect(source).toContain("instance.connection-recovery.auth.steps");
    expect(source).toContain("instance.connection-recovery.network.steps");
    expect(source).toContain("instance.connection-recovery.permission.steps");
    expect(source).toContain("instance.connection-recovery.timeout.steps");
    expect(source).toContain("instance.connection-recovery.tls.steps");
    expect(source).toContain("instance.connection-recovery.unsupported.steps");
    expect(source).toContain("instance.connection-recovery.unknown.steps");
  });

  test("links to official connection docs instead of backend-only guidance", () => {
    const source = readFileSync(
      join(process.cwd(), "src/components/instance/ConnectionRecovery.tsx"),
      "utf-8"
    );
    const enUS = readFileSync(
      join(process.cwd(), "src/locales/en-US.json"),
      "utf-8"
    );

    expect(source).toContain(
      '"https://docs.bytebase.com/get-started/connect/overview?source=console"'
    );
    expect(source).toContain('target="_blank"');
    expect(source).toContain('rel="noreferrer"');
    expect(source).toContain("instance.connection-recovery.docs");
    expect(enUS).not.toContain("Check Bytebase backend logs");
    expect(enUS).not.toContain("Review required connection fields");
  });

  test("shows Cloud-specific network recovery in SaaS mode", async () => {
    mocks.isSaaSMode = true;
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(
        createElement(ConnectionRecovery, { category: "network_unreachable" })
      );
    });

    expect(container.textContent).toContain(
      "instance.connection-recovery.network.description-saas"
    );
    expect(container.textContent).toContain(
      "instance.connection-recovery.network.steps.firewall-saas"
    );
    expect(container.querySelector("a")).toHaveAttribute(
      "href",
      "https://docs.bytebase.com/get-started/cloud#prerequisites"
    );

    act(() => root.unmount());
  });

  test("shows deployment-specific network recovery when self-hosted", async () => {
    const container = document.createElement("div");
    const root = createRoot(container);

    await act(async () => {
      root.render(
        createElement(ConnectionRecovery, { category: "network_unreachable" })
      );
    });

    expect(container.textContent).toContain(
      "instance.connection-recovery.network.description-self-hosted"
    );
    expect(container.textContent).toContain(
      "instance.connection-recovery.network.steps.firewall-self-hosted"
    );
    expect(container.querySelector("a")).toHaveAttribute(
      "href",
      "https://docs.bytebase.com/get-started/connect/overview?source=console"
    );

    act(() => root.unmount());
  });
});
