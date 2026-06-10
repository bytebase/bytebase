import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { describe, expect, test } from "vitest";
import type { IdentityProvider } from "@/types/proto-es/v1/idp_service_pb";
import { IdpBrandIcon } from "./IdpBrandIcon";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);
  act(() => {
    root.render(element);
  });
  return container;
};

const oauth2Idp = (title: string, authUrl: string) =>
  ({
    name: "idps/test",
    title,
    config: { config: { case: "oauth2Config", value: { authUrl } } },
  }) as IdentityProvider;

const oidcIdp = (title: string, issuer: string) =>
  ({
    name: "idps/test",
    title,
    config: { config: { case: "oidcConfig", value: { issuer } } },
  }) as IdentityProvider;

const configlessIdp = (title: string) =>
  ({ name: "idps/test", title }) as IdentityProvider;

// The Google "G" carries its mandatory brand blue; GitHub is currentColor.
const GOOGLE_BLUE = "#4285F4";

describe("IdpBrandIcon", () => {
  test("resolves brand from the OAuth2 auth URL, not the title", () => {
    const container = renderIntoContainer(
      <IdpBrandIcon
        idp={oauth2Idp("Corp SSO", "https://github.com/login/oauth/authorize")}
      />
    );
    const svg = container.querySelector("svg");
    expect(svg).toBeTruthy();
    expect(svg?.getAttribute("fill")).toBe("currentColor");
  });

  test("resolves brand from the OIDC issuer", () => {
    const container = renderIntoContainer(
      <IdpBrandIcon
        idp={oidcIdp("Work account", "https://accounts.google.com")}
      />
    );
    expect(
      container.querySelector(`svg path[fill="${GOOGLE_BLUE}"]`)
    ).toBeTruthy();
  });

  test("does not trust the title when the config points elsewhere", () => {
    const container = renderIntoContainer(
      <IdpBrandIcon
        idp={oidcIdp("GitHub Enterprise via Okta", "https://corp.okta.com")}
      />
    );
    expect(container.querySelector("svg")).toBeNull();
  });

  test("falls back to the title when no config URL is available", () => {
    const container = renderIntoContainer(
      <IdpBrandIcon idp={configlessIdp("GitLab")} />
    );
    expect(container.querySelector('svg path[fill="#FC6D26"]')).toBeTruthy();
  });

  test("renders nothing for unrecognized providers", () => {
    const container = renderIntoContainer(
      <IdpBrandIcon idp={configlessIdp("Corp SSO")} />
    );
    expect(container.querySelector("svg")).toBeNull();
  });
});
