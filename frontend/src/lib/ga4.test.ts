import { afterEach, describe, expect, test, vi } from "vitest";
import { initializeGA4 } from "./ga4";

const ga4ScriptSelector = "script#bytebase-ga4-tag";

declare global {
  interface Window {
    dataLayer?: unknown[][];
    gtag?: (...args: unknown[]) => void;
  }
}

afterEach(() => {
  vi.unstubAllEnvs();
  document.querySelector(ga4ScriptSelector)?.remove();
  delete window.dataLayer;
  delete window.gtag;
  window.history.replaceState(null, "", "/");
});

describe("initializeGA4", () => {
  test("does not load GA4 without a measurement ID", () => {
    initializeGA4();

    expect(document.querySelector(ga4ScriptSelector)).toBeNull();
    expect(window.dataLayer).toBeUndefined();
    expect(window.gtag).toBeUndefined();
  });

  test("loads the configured GA4 property", () => {
    vi.stubEnv("BB_GA4_MEASUREMENT_ID", "G-TEST");

    initializeGA4();

    const script = document.querySelector<HTMLScriptElement>(ga4ScriptSelector);
    expect(script?.async).toBe(true);
    expect(script?.src).toBe(
      "https://www.googletagmanager.com/gtag/js?id=G-TEST"
    );
    expect(window.dataLayer).toHaveLength(2);
    expect(window.dataLayer?.[0][0]).toBe("js");
    expect(window.dataLayer?.[1]).toEqual([
      "config",
      "G-TEST",
      {
        page_location: `${window.location.origin}/`,
        page_path: "/",
      },
    ]);
  });

  test("sanitizes the initial page view URL", () => {
    vi.stubEnv("BB_GA4_MEASUREMENT_ID", "G-TEST");

    window.history.replaceState(
      null,
      "",
      "/oauth/callback?code=secret&state=token#fragment"
    );

    initializeGA4();

    expect(window.dataLayer?.[1]).toEqual([
      "config",
      "G-TEST",
      {
        page_location: `${window.location.origin}/oauth/callback`,
        page_path: "/oauth/callback",
      },
    ]);
  });
});
