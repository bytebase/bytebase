import { beforeEach, describe, expect, test, vi } from "vitest";
import { type SchemaInfo, schemas } from "./gen/openapi-index";

const mocks = vi.hoisted(() => ({
  fetch: vi.fn(),
  getEndpointPath: vi.fn(),
  getRequestSchema: vi.fn(),
  getSchema: vi.fn(),
  refreshTokens: vi.fn(),
}));

vi.mock("@/connect/refreshToken", () => ({
  refreshTokens: mocks.refreshTokens,
}));

vi.mock("./searchApi", () => ({
  getEndpointPath: mocks.getEndpointPath,
  getRequestSchema: mocks.getRequestSchema,
  getSchema: mocks.getSchema,
}));

import { __testOnly, callApi } from "./callApi";

const createResponse = ({
  status,
  body,
  rejectJSON = false,
}: {
  status: number;
  body?: unknown;
  rejectJSON?: boolean;
}) =>
  ({
    status,
    json: rejectJSON
      ? vi.fn().mockRejectedValue(new Error("invalid json"))
      : vi.fn().mockResolvedValue(body),
  }) as never;

const getRequestBody = (): unknown => {
  const request = mocks.fetch.mock.calls.at(-1)?.[1];
  const body = request && "body" in request ? request.body : undefined;
  return JSON.parse(typeof body === "string" ? body : "{}");
};

describe("callApi", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", mocks.fetch);
    mocks.fetch.mockReset();
    mocks.getEndpointPath.mockReset();
    mocks.getRequestSchema.mockReset();
    mocks.getSchema.mockReset();
    mocks.refreshTokens.mockReset();
    mocks.getEndpointPath.mockReturnValue("/bytebase.v1.SQLService/Query");
    mocks.getRequestSchema.mockReturnValue(undefined);
    mocks.getSchema.mockReturnValue(undefined);
  });

  test("refreshes once and retries unauthenticated requests", async () => {
    mocks.fetch
      .mockResolvedValueOnce(
        createResponse({
          status: 401,
          body: { message: "access token not found" },
        })
      )
      .mockResolvedValueOnce(
        createResponse({
          status: 200,
          body: { rows: [] },
        })
      );
    mocks.refreshTokens.mockResolvedValue(undefined);

    const result = JSON.parse(
      await callApi({
        operationId: "SQLService/Query",
        body: { statement: "SELECT 1" },
      })
    );

    expect(result).toEqual({
      status: 200,
      response: { rows: [] },
    });
    expect(mocks.refreshTokens).toHaveBeenCalledTimes(1);
    expect(mocks.fetch).toHaveBeenCalledTimes(2);
    expect(mocks.fetch).toHaveBeenNthCalledWith(
      1,
      expect.stringContaining("/bytebase.v1.SQLService/Query"),
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
          "Connect-Protocol-Version": "1",
        },
        body: JSON.stringify({ statement: "SELECT 1" }),
      })
    );
  });

  test("does not recurse on the refresh endpoint", async () => {
    mocks.getEndpointPath.mockReturnValue("/bytebase.v1.AuthService/Refresh");
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 401,
        body: { message: "refresh token expired" },
      })
    );

    const result = JSON.parse(
      await callApi({
        operationId: "AuthService/Refresh",
      })
    );

    expect(result).toEqual({
      status: 401,
      error: "refresh token expired",
      response: { message: "refresh token expired" },
    });
    expect(mocks.refreshTokens).not.toHaveBeenCalled();
    expect(mocks.fetch).toHaveBeenCalledTimes(1);
  });

  test("preserves the original error shape when refresh fails", async () => {
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 401,
        body: { message: "access token not found" },
      })
    );
    mocks.refreshTokens.mockRejectedValue(new Error("refresh failed"));

    const result = JSON.parse(
      await callApi({
        operationId: "SQLService/Query",
      })
    );

    expect(result).toEqual({
      status: 401,
      error: "access token not found",
      response: { message: "access token not found" },
    });
    expect(mocks.refreshTokens).toHaveBeenCalledTimes(1);
    expect(mocks.fetch).toHaveBeenCalledTimes(1);
  });

  test("preserves non-json error handling", async () => {
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 500,
        rejectJSON: true,
      })
    );

    const result = JSON.parse(
      await callApi({
        operationId: "SQLService/Query",
      })
    );

    expect(result).toEqual({
      status: 500,
      error: "HTTP 500",
      response: null,
    });
    expect(mocks.refreshTokens).not.toHaveBeenCalled();
  });

  test("coerces nested byte fields to base64 before sending the request", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.WorksheetService/CreateWorksheet"
    );
    mocks.getRequestSchema.mockReturnValue(
      schemas["bytebase.v1.CreateWorksheetRequest"]
    );
    mocks.getSchema.mockImplementation(
      (schemaName: string) => schemas[schemaName]
    );
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 200,
        body: { name: "projects/demo/worksheets/1" },
      })
    );

    await callApi({
      operationId: "WorksheetService/CreateWorksheet",
      body: {
        parent: "projects/demo",
        worksheet: {
          content: "select 1;",
          title: "demo",
        },
      },
    });

    expect(getRequestBody()).toEqual({
      parent: "projects/demo",
      worksheet: {
        content: "c2VsZWN0IDE7",
        title: "demo",
      },
    });
  });

  test("coerces byte fields inside arrays using the real batch create sheets schema", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.SheetService/BatchCreateSheets"
    );
    mocks.getRequestSchema.mockReturnValue(
      schemas["bytebase.v1.BatchCreateSheetsRequest"]
    );
    mocks.getSchema.mockImplementation(
      (schemaName: string) => schemas[schemaName]
    );
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 200,
        body: { sheets: [] },
      })
    );

    await callApi({
      operationId: "SheetService/BatchCreateSheets",
      body: {
        parent: "projects/demo",
        requests: [
          {
            parent: "projects/demo",
            sheet: {
              content: "select 1;",
              title: "sheet one",
            },
          },
          {
            parent: "projects/demo",
            sheet: {
              content: "select 2;",
              title: "sheet two",
            },
          },
        ],
      },
    });

    expect(getRequestBody()).toEqual({
      parent: "projects/demo",
      requests: [
        {
          parent: "projects/demo",
          sheet: {
            content: "c2VsZWN0IDE7",
            title: "sheet one",
          },
        },
        {
          parent: "projects/demo",
          sheet: {
            content: "c2VsZWN0IDI7",
            title: "sheet two",
          },
        },
      ],
    });
  });

  test("coerces byte fields through external oneof schemas in batch deparse requests", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.CelService/BatchDeparse"
    );
    mocks.getRequestSchema.mockReturnValue(
      schemas["bytebase.v1.BatchDeparseRequest"]
    );
    mocks.getSchema.mockImplementation(
      (schemaName: string) => schemas[schemaName]
    );
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 200,
        body: { expressions: [] },
      })
    );

    await callApi({
      operationId: "CelService/BatchDeparse",
      body: {
        expressions: [
          {
            id: "1",
            constExpr: {
              bytesValue: "hello",
            },
          },
        ],
      },
    });

    expect(getRequestBody()).toEqual({
      expressions: [
        {
          id: "1",
          constExpr: {
            bytesValue: "aGVsbG8=",
          },
        },
      ],
    });
  });

  test("coerces byte fields through top-level oneof sasl config branches", async () => {
    mocks.getEndpointPath.mockReturnValue(
      "/bytebase.v1.InstanceService/AddDataSource"
    );
    mocks.getRequestSchema.mockReturnValue(
      schemas["bytebase.v1.AddDataSourceRequest"]
    );
    mocks.getSchema.mockImplementation(
      (schemaName: string) => schemas[schemaName]
    );
    mocks.fetch.mockResolvedValue(
      createResponse({
        status: 200,
        body: { name: "instances/demo/dataSources/readonly" },
      })
    );

    await callApi({
      operationId: "InstanceService/AddDataSource",
      body: {
        name: "instances/demo",
        dataSource: {
          saslConfig: {
            krbConfig: {
              primary: "postgres",
              realm: "EXAMPLE.COM",
              keytab: "keytab-bytes",
            },
          },
        },
      },
    });

    expect(getRequestBody()).toEqual({
      name: "instances/demo",
      dataSource: {
        saslConfig: {
          krbConfig: {
            primary: "postgres",
            realm: "EXAMPLE.COM",
            keytab: "a2V5dGFiLWJ5dGVz",
          },
        },
      },
    });
  });

  test("coerces byte fields inside map values through the helper path", () => {
    const schema: SchemaInfo = {
      type: "object",
      description: "test schema for map coercion",
      properties: [
        {
          name: "entries",
          type: "object",
          additionalProperties: {
            name: "value",
            type: "test.Entry",
          },
        },
      ],
    };

    mocks.getSchema.mockImplementation((schemaName: string) => {
      if (schemaName === "test.Entry") {
        return {
          type: "object",
          description: "test entry schema",
          properties: [
            {
              name: "content",
              type: "string",
              format: "byte",
            },
          ],
        };
      }
      return schemas[schemaName];
    });

    expect(
      __testOnly.coerceRequestBody(
        {
          entries: {
            alpha: { content: "select 1;" },
            beta: { content: "select 2;" },
          },
        },
        schema
      )
    ).toEqual({
      entries: {
        alpha: { content: "c2VsZWN0IDE7" },
        beta: { content: "c2VsZWN0IDI7" },
      },
    });
  });
});
