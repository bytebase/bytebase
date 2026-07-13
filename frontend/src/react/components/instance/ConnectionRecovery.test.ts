import { readFileSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, test } from "vitest";
import { normalizeConnectionFailureCategory } from "./ConnectionRecovery";

describe("ConnectionRecovery", () => {
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
      join(
        process.cwd(),
        "src/react/components/instance/ConnectionRecovery.tsx"
      ),
      "utf-8"
    );

    expect(source).toContain('from "@/react/components/ui/alert"');
    expect(source).toContain("<Alert");
    expect(source).toContain('variant="warning"');
    expect(source).not.toContain("border-control-border bg-control-bg");
    expect(source).not.toContain("toLowerCase");
  });

  test("shows category-specific recovery steps", () => {
    const source = readFileSync(
      join(
        process.cwd(),
        "src/react/components/instance/ConnectionRecovery.tsx"
      ),
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
      join(
        process.cwd(),
        "src/react/components/instance/ConnectionRecovery.tsx"
      ),
      "utf-8"
    );
    const enUS = readFileSync(
      join(process.cwd(), "src/locales/en-US.json"),
      "utf-8"
    );

    expect(source).toContain(
      'href="https://docs.bytebase.com/get-started/connect/overview?source=console"'
    );
    expect(source).toContain('target="_blank"');
    expect(source).toContain('rel="noreferrer"');
    expect(source).toContain("instance.connection-recovery.docs");
    expect(enUS).not.toContain("Check Bytebase backend logs");
    expect(enUS).not.toContain("Review required connection fields");
  });
});
