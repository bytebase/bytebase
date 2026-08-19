import { render } from "@testing-library/react";
import { describe, expect, test } from "vitest";
import {
  highlightHtmlText,
  renderRowFieldsWithSearchMatches,
} from "./detail-panel-search";

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

  test("preserves named row fields and the global active match order", () => {
    const result = renderRowFieldsWithSearchMatches(
      [
        { columnName: "name", value: "primary name" },
        { columnName: "display_name", value: "Ada" },
      ],
      "name",
      2
    );
    const view = render(
      <>
        <span>{result.fields[0]?.columnName}</span>
        <span>{result.fields[0]?.value}</span>
        <span>{result.fields[1]?.columnName}</span>
        <span>{result.fields[1]?.value}</span>
      </>
    );

    const matches = view.container.querySelectorAll("mark");
    expect(result.count).toBe(3);
    expect(matches).toHaveLength(3);
    expect(matches[2]).toHaveAttribute(
      "data-detail-search-active-match",
      "true"
    );
  });
});
