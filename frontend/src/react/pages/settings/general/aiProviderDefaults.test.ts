import { describe, expect, test } from "vitest";
import { AISetting_Provider } from "@/types/proto-es/v1/setting_service_pb";
import { PROVIDER_DEFAULTS } from "./aiProviderDefaults";

describe("PROVIDER_DEFAULTS", () => {
  test("uses current default model IDs for built-in AI providers", () => {
    expect(PROVIDER_DEFAULTS[AISetting_Provider.OPEN_AI].model).toBe("gpt-5.5");
    expect(PROVIDER_DEFAULTS[AISetting_Provider.AZURE_OPENAI].model).toBe(
      "gpt-5.5"
    );
    expect(PROVIDER_DEFAULTS[AISetting_Provider.GEMINI].model).toBe(
      "gemini-3.5-flash"
    );
    expect(PROVIDER_DEFAULTS[AISetting_Provider.CLAUDE].model).toBe(
      "claude-sonnet-5"
    );
  });
});
