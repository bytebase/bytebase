import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  fetch: vi.fn(),
  getEndpointPath: vi.fn(),
  refreshTokens: vi.fn(),
}));

vi.mock("@/connect/refreshToken", () => ({
  refreshTokens: mocks.refreshTokens,
}));

vi.mock("./searchApi", () => ({
  getEndpointPath: mocks.getEndpointPath,
}));

import { callApi } from "./callApi";

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

describe("callApi", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", mocks.fetch);
    mocks.fetch.mockReset();
    mocks.getEndpointPath.mockReset();
    mocks.refreshTokens.mockReset();
    mocks.getEndpointPath.mockReturnValue("/bytebase.v1.SQLService/Query");
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
});
