import { Code } from "@connectrpc/connect";
import { describe, expect, it } from "vitest";
import { parseAgentRpcError } from "./apiRpcError";

describe("parseAgentRpcError", () => {
  it("uses Connect's error decoder without accepting duplicate codes", () => {
    expect(
      parseAgentRpcError('{"code":"permission_denied","message":"Denied"}', 403)
    ).toMatchObject({ code: Code.PermissionDenied, rawMessage: "Denied" });
    expect(
      parseAgentRpcError(
        '{"code":"permission_denied","code":"permission_denied"}',
        403
      )
    ).toBeUndefined();
  });

  it("can decode a client error without asserting an observed HTTP status", () => {
    expect(parseAgentRpcError('{"code":"unavailable"}')).toMatchObject({
      code: Code.Unavailable,
    });
    expect(parseAgentRpcError('{"code":"unavailable"}', 500)).toBeUndefined();
  });

  it.each([
    ["invalid JSON", "{"],
    ["HTTP/code mismatch", '{"code":"failed_precondition"}'],
    ["unknown code", '{"code":"unrecognized"}'],
    ["invalid details", '{"code":"permission_denied","details":{}}'],
  ])("rejects %s", (_description, response) => {
    expect(parseAgentRpcError(response, 403)).toBeUndefined();
  });
});
