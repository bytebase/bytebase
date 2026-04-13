import { describe, expect, test } from "vitest";
import {
  RESIZE_POINTER_MEDIA_QUERY,
  supportsWindowBorderResize,
} from "./resize-capability";

describe("supportsWindowBorderResize", () => {
  test("uses any-pointer detection so hybrid devices keep window resizing", () => {
    let receivedQuery = "";

    const supported = supportsWindowBorderResize((query) => {
      receivedQuery = query;
      return { matches: true };
    });

    expect(receivedQuery).toBe("(any-hover: hover) and (any-pointer: fine)");
    expect(RESIZE_POINTER_MEDIA_QUERY).toBe(receivedQuery);
    expect(supported).toBe(true);
  });
});
