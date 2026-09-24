// OAuth2 consent (/oauth2/consent): the page an MCP client sends its user to
// before the client is granted a session.

import { expect, test } from "@playwright/test";
import { loadTestEnv } from "../framework/env";

// The card's design: centered in the viewport at one width (max-w-2xl), and
// narrower only when the viewport minus the page gutters (px-4) cannot hold it.
const CARD_WIDTH = 672;
const GUTTER = 16;

const VIEWPORTS = [
  { width: 390, height: 844 },
  { width: 700, height: 900 },
  { width: 900, height: 1000 },
  { width: 1024, height: 768 },
  { width: 1280, height: 800 },
  { width: 1440, height: 900 },
  { width: 1920, height: 1080 },
];

const REDIRECT_URI = "http://localhost:33418/callback";

let consentURL: string;

test.beforeAll(async () => {
  const env = loadTestEnv();
  // No teardown: registration has no delete endpoint, and a client nobody
  // authorizes grants nothing.
  const { client_id } = await env.api.registerOAuth2Client(
    "E2E consent layout",
    [REDIRECT_URI]
  );
  consentURL = `${env.baseURL}/oauth2/consent?${new URLSearchParams({
    client_id,
    redirect_uri: REDIRECT_URI,
    state: "e2e",
    // The page only requires a challenge; the authorize POST verifies it, and
    // this spec never sends one.
    code_challenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
    code_challenge_method: "S256",
  })}`;
});

test.describe("OAuth2 consent card", () => {
  for (const viewport of VIEWPORTS) {
    test(`keeps its width unless the viewport is narrower, at ${viewport.width}px`, async ({
      page,
    }) => {
      await page.setViewportSize(viewport);
      await page.goto(consentURL);
      const card = page.getByTestId("oauth2-consent-card");
      // Every branch that has loaded its client renders a level-1 heading.
      await expect(card.getByRole("heading", { level: 1 })).toBeVisible();

      // clientWidth rather than the requested width, so a classic scrollbar
      // is not counted as room.
      const measured = await card.evaluate((el) => {
        const rect = el.getBoundingClientRect();
        return {
          viewport: document.documentElement.clientWidth,
          left: rect.left,
          width: rect.width,
        };
      });
      const expected = Math.min(CARD_WIDTH, measured.viewport - 2 * GUTTER);
      expect(
        Math.abs(measured.width - expected),
        `card ${measured.width}px in a ${measured.viewport}px viewport`
      ).toBeLessThanOrEqual(1);
      expect(
        Math.abs(measured.left - (measured.viewport - measured.width) / 2),
        `card starts at ${measured.left}px`
      ).toBeLessThanOrEqual(1);
    });
  }
});
