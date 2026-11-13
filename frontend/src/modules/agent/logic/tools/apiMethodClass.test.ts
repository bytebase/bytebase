import { describe, expect, it } from "vitest";
import { MCPMethodClass } from "@/types/proto-es/v1/annotation_pb";
import {
  apiCallNeedsConfirmation,
  getApiAuthorization,
  getApiMethodClass,
  getApiMethodDescriptor,
} from "./apiMethodClass";
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

describe("getApiMethodDescriptor", () => {
  it("returns the request descriptor for short and full operation IDs", () => {
    const method = getApiMethodDescriptor("InstanceService/UpdateInstance");
    expect(method?.input.typeName).toBe("bytebase.v1.UpdateInstanceRequest");
    expect(
      getApiMethodDescriptor("bytebase.v1.InstanceService.UpdateInstance")
    ).toBe(method);
  });

  it("does not infer a descriptor for an unknown operation", () => {
    expect(
      getApiMethodDescriptor("InstanceService/DoesNotExist")
    ).toBeUndefined();
  });
});

describe("getApiAuthorization", () => {
  it("reports declared IAM permission for a target API", () => {
    expect(getApiAuthorization("ProjectService/UpdateProject")).toEqual({
      authMethod: "IAM",
      permission: "bb.projects.update",
    });
  });

  it("does not claim a fixed permission for CUSTOM APIs", () => {
    expect(getApiAuthorization("DatabaseService/ListDatabases")).toEqual({
      authMethod: "CUSTOM",
    });
  });

  it("returns undefined for unknown operations", () => {
    expect(getApiAuthorization("ProjectService/DoesNotExist")).toBeUndefined();
  });

  it("has a declared permission for every indexed IAM API", () => {
    expect(
      endpoints
        .filter((endpoint) => {
          const authorization = getApiAuthorization(endpoint.operationId);
          return (
            authorization?.authMethod === "IAM" && !authorization.permission
          );
        })
        .map((endpoint) => endpoint.operationId)
    ).toEqual([]);
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

  it("requires confirmation for SettingService/GetSetting despite its READ class", () => {
    expect(getApiMethodClass("SettingService/GetSetting")).toBe(
      MCPMethodClass.READ
    );
    expect(apiCallNeedsConfirmation("SettingService/GetSetting")).toBe(true);
  });
});
