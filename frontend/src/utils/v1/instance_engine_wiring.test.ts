import { describe, expect, test } from "vitest";
import {
  Engine,
  EngineSchema,
} from "@/types/proto-es/v1/common_pb";
import {
  defaultPortForEngine,
  EngineIconPath,
} from "@/components/InstanceForm/constants";
import { engineNameV1, supportedEngineV1List } from "./instance";

describe("instance engine wiring", () => {
  test("supportedEngineV1List has no duplicates", () => {
    const supported = supportedEngineV1List();
    expect(new Set(supported).size).toBe(supported.length);
  });

  test("every supported engine has name, port and icon mapping", () => {
    for (const engine of supportedEngineV1List()) {
      expect(engine).not.toBe(Engine.ENGINE_UNSPECIFIED);
      expect(engineNameV1(engine)).not.toBe("");
      expect(() => defaultPortForEngine(engine)).not.toThrow();
      expect(EngineIconPath[engine]).toBeTruthy();
    }
  });

  test("supported engines are valid enum values", () => {
    const validEngines = new Set(
      EngineSchema.value.values
        .map((value) => value.number)
        .filter((value) => value !== Engine.ENGINE_UNSPECIFIED)
    );

    for (const engine of supportedEngineV1List()) {
      expect(validEngines.has(engine)).toBe(true);
    }
  });
});
