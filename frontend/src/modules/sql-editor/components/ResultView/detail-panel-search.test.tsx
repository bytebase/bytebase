import { describe, expect, test } from "vitest";
import { highlightHtmlText } from "./detail-panel-search";

describe("detail panel search", () => {
  test("highlights matches inside generated HTML without removing token markup", () => {
    const result = highlightHtmlText(
      '<span class="json-key">"raw_dump"</span>: <span class="json-string">"CREATE TABLE users"</span>',
      "create",
      0
    );

    expect(result.count).toBe(1);
    expect(result.html).toContain('class="json-key"');
    expect(result.html).toContain('class="json-string"');
    expect(result.html).toContain("<mark");
    expect(result.html).toContain("CREATE");
  });
});
