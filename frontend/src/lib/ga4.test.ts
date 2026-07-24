import { afterEach, describe, expect, test } from "vitest";
import { initializeGA4 } from "./ga4";

const ga4ScriptSelector = "script#bytebase-ga4-tag";

declare global {
  interface Window {
    dataLayer?: unknown[][];
    gtag?: (...args: unknown[]) => void;
  }
}

afterEach(() => {
  document.querySelector(ga4ScriptSelector)?.remove();
  delete window.dataLayer;
  delete window.gtag;
  window.history.replaceState(null, "", "/");
});

describe("initializeGA4", () => {
  test("does not load GA4 outside SaaS mode", () => {
    initializeGA4(false);

    expect(document.querySelector(ga4ScriptSelector)).toBeNull();
    expect(window.dataLayer).toBeUndefined();
    expect(window.gtag).toBeUndefined();
  });

  test("loads the shared GA4 property in SaaS mode", () => {
    initializeGA4(true);

    const script = document.querySelector<HTMLScriptElement>(ga4ScriptSelector);
    expect(script?.async).toBe(true);
    expect(script?.src).toBe(
      "https://www.googletagmanager.com/gtag/js?id=G-4BZ4JH7449"
    );
    expect(window.dataLayer).toHaveLength(2);
    expect(window.dataLayer?.[0][0]).toBe("js");
    expect(window.dataLayer?.[1]).toEqual([
      "config",
      "G-4BZ4JH7449",
      {
        page_location: `${window.location.origin}/`,
        page_path: "/",
      },
    ]);
  });

  test("sanitizes the initial page view URL", () => {
    window.history.replaceState(
      null,
      "",
      "/oauth/callback?code=secret&state=token#fragment"
    );

    initializeGA4(true);

    expect(window.dataLayer?.[1]).toEqual([
      "config",
      "G-4BZ4JH7449",
      {
        page_location: `${window.location.origin}/oauth/callback`,
        page_path: "/oauth/callback",
      },
    ]);
  });
});
