import { beforeEach, describe, expect, it, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";

vi.mock("@/plugins/i18n", () => ({
  t: (key: string) => key,
}));

vi.mock("@/store", () => ({
  useSubscriptionV1Store: () => ({ currentPlan: 0 }),
}));

vi.mock("@/types", () => ({
  isValidInstanceName: () => true,
  languageOfEngineV1: () => "sql",
  unknownInstance: () => ({ title: "Unknown" }),
}));

describe("instanceV1HasExtraParameters", () => {
  beforeEach(() => {
    vi.resetModules();
  });

  it("supports TiDB extra connection parameters", async () => {
    const { instanceV1HasExtraParameters } = await import("./instance");
    expect(instanceV1HasExtraParameters(Engine.TIDB)).toBe(true);
  });
});
