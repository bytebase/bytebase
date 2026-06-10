import { describe, expect, test, vi } from "vitest";

// util.ts transitively pulls in i18n-aware modules; stub the i18n surface so
// importing the highlight helper doesn't require the full i18n runtime.
vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (k: string) => k }),
  initReactI18next: { type: "3rdParty", init: () => {} },
}));

import { getHighlightHTMLByRegExp } from "./util";

describe("getHighlightHTMLByRegExp", () => {
  test("highlights a plain keyword", () => {
    const html = getHighlightHTMLByRegExp("SELECT 1", "select");
    expect(html).toContain('<b class="text-accent">SELECT</b>');
  });

  test("highlights a keyword containing a double quote (HTML-escaped target)", () => {
    // Regression: `"public"` must still highlight even though the target's
    // quotes are HTML-escaped to &quot; before matching.
    const html = getHighlightHTMLByRegExp('FROM "public".db', [
      '"public".db',
    ] as string[]);
    // The matched substring (incl. the quoted identifier) is wrapped in <b>;
    // before the fix this token was not highlighted at all.
    expect(html).toContain("<b");
    expect(html).toContain("public");
  });

  test("highlights multiple whitespace-split tokens including a quoted one", () => {
    const html = getHighlightHTMLByRegExp('SELECT * FROM "public".db', [
      "SELECT",
      "*",
      '"public".db',
    ]);
    // All three tokens get wrapped.
    expect((html.match(/<b /g) ?? []).length).toBeGreaterThanOrEqual(3);
  });
});
