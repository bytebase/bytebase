import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";
import {
  exceedsSheetPreviewLimit,
  isCappedSheetPreview,
  SHEET_PREVIEW_CHARACTER_LIMIT,
} from "./sheet";

describe("exceedsSheetPreviewLimit", () => {
  test("matches the API's character boundary", () => {
    const statement = "a".repeat(SHEET_PREVIEW_CHARACTER_LIMIT);
    expect(exceedsSheetPreviewLimit(statement)).toBe(false);
    expect(exceedsSheetPreviewLimit(`${statement}a`)).toBe(true);
  });

  test("does not treat a multibyte sheet as truncated solely because of its byte size", () => {
    const statement = "表".repeat(1_000_000);
    expect(new TextEncoder().encode(statement).byteLength).toBeGreaterThan(
      2 * 1024 * 1024
    );
    expect(exceedsSheetPreviewLimit(statement)).toBe(false);
  });

  test("counts supplementary Unicode characters once", () => {
    const statement = `${"a".repeat(SHEET_PREVIEW_CHARACTER_LIMIT - 1)}😀`;
    expect(exceedsSheetPreviewLimit(statement)).toBe(false);
    expect(exceedsSheetPreviewLimit(`${statement}a`)).toBe(true);
  });
});

test("recognizes only incomplete previews that reach the API cutoff", () => {
  const content = new TextEncoder().encode(
    "a".repeat(SHEET_PREVIEW_CHARACTER_LIMIT)
  );
  const sheet = (size: bigint) =>
    create(SheetSchema, { content, contentSize: size });
  expect(isCappedSheetPreview(sheet(BigInt(content.length + 1)))).toBe(true);
  expect(isCappedSheetPreview(sheet(BigInt(content.length)))).toBe(false);
  expect(
    isCappedSheetPreview(
      create(SheetSchema, {
        content: new TextEncoder().encode("short"),
        contentSize: 100n,
      })
    )
  ).toBe(false);
});
