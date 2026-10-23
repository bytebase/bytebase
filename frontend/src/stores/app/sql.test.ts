// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import type { ContextValues } from "@connectrpc/connect";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import { ExportDelivery } from "@/types/proto-es/v1/common_pb";
import {
  ExportRequestSchema,
  ExportResponseSchema,
} from "@/types/proto-es/v1/sql_service_pb";
import { createSQLSlice } from "./sql";

const mocks = vi.hoisted(() => ({ export: vi.fn() }));

vi.mock("@/api", () => ({
  sqlServiceClientConnect: { export: mocks.export },
}));

const slice = () =>
  createSQLSlice((() => {}) as never, (() => ({})) as never, {} as never);

const contextValuesOfLastCall = () =>
  (mocks.export.mock.calls[0][1] as { contextValues: ContextValues })
    .contextValues;

describe("exportData", () => {
  beforeEach(() => {
    mocks.export.mockReset();
  });

  test("returns local content and reports errors normally", async () => {
    const content = new Uint8Array([1, 2]);
    mocks.export.mockResolvedValue(create(ExportResponseSchema, { content }));

    const result = await slice().exportData(
      create(ExportRequestSchema, { delivery: ExportDelivery.LOCAL })
    );

    expect(result).toEqual({ content, uri: "" });
    expect(contextValuesOfLastCall().get(silentContextKey)).toBe(false);
  });

  test("returns the delivered folder URI and leaves failures to the export result", async () => {
    const uri = "https://contoso.sharepoint.com/sites/Finance/Exports/20261022";
    mocks.export.mockResolvedValue(create(ExportResponseSchema, { uri }));

    const result = await slice().exportData(
      create(ExportRequestSchema, { delivery: ExportDelivery.SHAREPOINT })
    );

    expect(result).toEqual({ content: new Uint8Array(), uri });
    expect(contextValuesOfLastCall().get(silentContextKey)).toBe(true);
  });
});
