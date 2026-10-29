import { describe, expect, it } from "vitest";
import { MCPMethodClass } from "@/types/proto-es/v1/annotation_pb";
import { apiCallNeedsConfirmation, getApiMethodClass } from "./apiMethodClass";
import { endpoints } from "./gen/openapi-index";

describe("getApiMethodClass", () => {
  it.each([
    ["ProjectService/GetProject", MCPMethodClass.READ],
    ["ProjectService/UpdateProject", MCPMethodClass.WRITE],
    ["AuthService/Login", MCPMethodClass.FORBIDDEN],
    ["AIService/Chat", MCPMethodClass.EXCLUDED],
  ] as const)(
    "returns the descriptor class for %s",
    (operationId, expected) => {
      expect(getApiMethodClass(operationId)).toBe(expected);
    }
  );

  it("resolves both short and full operation IDs", () => {
    expect(getApiMethodClass("bytebase.v1.ProjectService.GetProject")).toBe(
      MCPMethodClass.READ
    );
  });

  it("does not classify an unknown operation as a read", () => {
    expect(getApiMethodClass("ProjectService/DoesNotExist")).toBeUndefined();
  });

  it("classifies every generated OpenAPI endpoint", () => {
    const unclassified = endpoints
      .filter(
        (endpoint) => getApiMethodClass(endpoint.operationId) === undefined
      )
      .map((endpoint) => endpoint.operationId);
    expect(unclassified).toEqual([]);
  });
});

describe("apiCallNeedsConfirmation", () => {
  it("allows only verified read methods to run without confirmation", () => {
    expect(apiCallNeedsConfirmation("ProjectService/GetProject")).toBe(false);
    expect(apiCallNeedsConfirmation("ProjectService/UpdateProject")).toBe(true);
    expect(apiCallNeedsConfirmation("AuthService/Login")).toBe(true);
    expect(apiCallNeedsConfirmation("AIService/Chat")).toBe(true);
    expect(apiCallNeedsConfirmation("ProjectService/DoesNotExist")).toBe(true);
  });

  it("requires confirmation for SQLService/Query despite its READ class", () => {
    expect(getApiMethodClass("SQLService/Query")).toBe(MCPMethodClass.READ);
    expect(apiCallNeedsConfirmation("SQLService/Query")).toBe(true);
  });
});
