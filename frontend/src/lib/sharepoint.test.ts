import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { SharePointLocationSchema } from "@/types/proto-es/v1/project_service_pb";
import { formatSharePointLocation } from "./sharepoint";

describe("formatSharePointLocation", () => {
  test("joins site, library, and folder", () => {
    expect(
      formatSharePointLocation(
        create(SharePointLocationSchema, {
          siteName: "Finance",
          libraryName: "Documents",
          folderPath: "Exports/2026",
        })
      )
    ).toBe("Finance / Documents / Exports/2026");
  });

  test("omits the folder for a library root", () => {
    expect(
      formatSharePointLocation(
        create(SharePointLocationSchema, {
          siteName: "Finance",
          libraryName: "Documents",
        })
      )
    ).toBe("Finance / Documents");
  });
});
