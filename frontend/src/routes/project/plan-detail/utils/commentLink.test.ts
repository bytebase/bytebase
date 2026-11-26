import { describe, expect, test } from "vitest";
import { commentFragmentOf, commentNameOfHash } from "./commentLink";

const issue = "projects/shop/issues/701";
const comment = `${issue}/issueComments/2f1c9a52`;

describe("commentFragmentOf", () => {
  test("is the last two segments of the comment's name", () => {
    expect(commentFragmentOf(comment)).toBe("issueComments/2f1c9a52");
  });

  test("is undefined for a name that is not a comment's", () => {
    expect(commentFragmentOf(issue)).toBeUndefined();
    expect(commentFragmentOf(`${issue}/issueComments/`)).toBeUndefined();
  });
});

describe("commentNameOfHash", () => {
  test("joins the plan's issue and the fragment", () => {
    expect(commentNameOfHash("#issueComments/2f1c9a52", issue)).toBe(comment);
  });

  test("round-trips with commentFragmentOf", () => {
    expect(commentNameOfHash(`#${commentFragmentOf(comment)}`, issue)).toBe(
      comment
    );
  });

  test("ignores other fragments", () => {
    expect(commentNameOfHash("", issue)).toBeUndefined();
    expect(commentNameOfHash("#plan-phase-review", issue)).toBeUndefined();
    expect(commentNameOfHash("#issueComments/", issue)).toBeUndefined();
    expect(commentNameOfHash("#issueComments/a/b", issue)).toBeUndefined();
    expect(commentNameOfHash("#issueComments/%E0%A4%A", issue)).toBeUndefined();
  });

  test("names nothing on a plan without an issue", () => {
    expect(
      commentNameOfHash("#issueComments/2f1c9a52", undefined)
    ).toBeUndefined();
  });
});
