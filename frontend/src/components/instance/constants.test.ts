import { describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { effectivePortForEngine } from "./constants";

vi.mock("@/utils", () => ({
  supportedEngineV1List: () => [],
}));

describe("effectivePortForEngine", () => {
  test("uses the engine default when the editable port is blank", () => {
    expect(effectivePortForEngine(Engine.MYSQL, "", true)).toBe("3306");
  });

  test("preserves explicit custom ports", () => {
    expect(effectivePortForEngine(Engine.MYSQL, "3307", true)).toBe("3307");
  });

  test("uses the engine default for MongoDB non-SRV data sources", () => {
    expect(effectivePortForEngine(Engine.MONGODB, "", false)).toBe("27017");
  });

  test("preserves blank ports for MongoDB SRV data sources", () => {
    expect(effectivePortForEngine(Engine.MONGODB, "", true)).toBe("");
  });
});
