// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import { SettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import { readSemanticTypeSetting } from "./semanticTypeSetting";

const mocks = vi.hoisted(() => ({ getSetting: vi.fn() }));

vi.mock("@/api", () => ({
  settingServiceClientConnect: { getSetting: mocks.getSetting },
}));

describe("readSemanticTypeSetting", () => {
  beforeEach(() => {
    mocks.getSetting.mockReset();
  });

  test("returns the freshly read setting unchanged", async () => {
    const setting = create(SettingSchema, {
      name: "settings/SEMANTIC_TYPES",
      value: { value: { case: "semanticType", value: { types: [] } } },
    });
    mocks.getSetting.mockResolvedValue(setting);

    const result = await readSemanticTypeSetting();

    expect(result.setting).toBe(setting);
    expect(result.configured).toBe(true);
    const [request, options] = mocks.getSetting.mock.calls[0];
    expect(request.name).toBe("settings/SEMANTIC_TYPES");
    expect(options.contextValues.get(silentContextKey)).toBe(true);
  });

  test("represents only a missing setting as known empty configuration", async () => {
    mocks.getSetting.mockRejectedValue(
      new ConnectError("missing", Code.NotFound)
    );

    const result = await readSemanticTypeSetting();

    expect(result).toEqual({
      setting: create(SettingSchema, {
        name: "settings/SEMANTIC_TYPES",
        value: { value: { case: "semanticType", value: { types: [] } } },
      }),
      configured: false,
    });
  });

  test.each([Code.PermissionDenied, Code.Unavailable, Code.Internal])(
    "preserves a failed read with code %s",
    async (code) => {
      const error = new ConnectError("read failed", code);
      mocks.getSetting.mockRejectedValue(error);

      await expect(readSemanticTypeSetting()).rejects.toBe(error);
    }
  );

  test("does not replace a malformed successful response with empty configuration", async () => {
    const setting = create(SettingSchema, { name: "settings/SEMANTIC_TYPES" });
    mocks.getSetting.mockResolvedValue(setting);

    const result = await readSemanticTypeSetting();

    expect(result.setting).toBe(setting);
    expect(result.configured).toBe(true);
    expect(result.setting.value).toBeUndefined();
  });
});
