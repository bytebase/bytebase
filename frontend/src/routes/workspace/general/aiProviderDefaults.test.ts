// @vitest-environment node
import { describe, expect, test } from "vitest";
import { AISetting_Provider } from "@/types/proto-es/v1/setting_service_pb";
import {
  getModelEndpoint,
  PROVIDER_DEFAULTS,
  PROVIDER_MODELS,
} from "./aiProviderDefaults";

describe("PROVIDER_DEFAULTS", () => {
  test("uses current default model IDs for built-in AI providers", () => {
    expect(PROVIDER_DEFAULTS[AISetting_Provider.OPEN_AI].model).toBe(
      "gpt-6.1-sol"
    );
    expect(PROVIDER_DEFAULTS[AISetting_Provider.AZURE_OPENAI].model).toBe(
      "gpt-6.1-sol"
    );
    expect(PROVIDER_DEFAULTS[AISetting_Provider.GEMINI].model).toBe(
      "gemini-3.8-flash"
    );
    expect(PROVIDER_DEFAULTS[AISetting_Provider.CLAUDE].model).toBe(
      "claude-sonnet-5-5"
    );
  });

  test("includes current supported models", () => {
    expect(
      PROVIDER_MODELS[AISetting_Provider.OPEN_AI].map((model) => model.value)
    ).toEqual(
      expect.arrayContaining([
        "gpt-6-astra",
        "gpt-6.1-sol",
        "gpt-6.1-terra",
        "gpt-6-luna",
      ])
    );
    expect(
      PROVIDER_MODELS[AISetting_Provider.AZURE_OPENAI].map(
        (model) => model.value
      )
    ).toEqual(
      expect.arrayContaining([
        "gpt-6-astra",
        "gpt-6.1-sol",
        "gpt-6.1-terra",
        "gpt-6-luna",
      ])
    );
    expect(
      PROVIDER_MODELS[AISetting_Provider.GEMINI].map((model) => model.value)
    ).toEqual(
      expect.arrayContaining([
        "gemini-4-argon",
        "gemini-3.8-flash",
        "gemini-3.6-flash",
        "gemini-3.5-flash-lite",
      ])
    );
    expect(
      PROVIDER_MODELS[AISetting_Provider.CLAUDE].map((model) => model.value)
    ).toEqual(
      expect.arrayContaining([
        "claude-fable-5-1",
        "claude-opus-5-5",
        "claude-sonnet-5-5",
        "claude-haiku-5-5",
      ])
    );
  });

  test("uses the Responses API for every OpenAI model", () => {
    for (const provider of [
      AISetting_Provider.OPEN_AI,
      AISetting_Provider.AZURE_OPENAI,
    ]) {
      for (const model of PROVIDER_MODELS[provider]) {
        expect(model.endpoint).toContain("/responses");
      }
    }
  });

  test("keeps each provider default aligned with its first model", () => {
    for (const provider of [
      AISetting_Provider.OPEN_AI,
      AISetting_Provider.AZURE_OPENAI,
      AISetting_Provider.GEMINI,
      AISetting_Provider.CLAUDE,
    ]) {
      expect(PROVIDER_DEFAULTS[provider].model).toBe(
        PROVIDER_MODELS[provider][0].value
      );
      expect(PROVIDER_DEFAULTS[provider].endpoint).toBe(
        PROVIDER_MODELS[provider][0].endpoint
      );
    }
  });

  test("preserves the Azure resource origin when changing models", () => {
    const provider = AISetting_Provider.AZURE_OPENAI;
    const luna = PROVIDER_MODELS[provider].find(
      (model) => model.value === "gpt-6-luna"
    );

    expect(luna).toBeDefined();
    expect(
      getModelEndpoint(provider, PROVIDER_DEFAULTS[provider].endpoint, luna!)
    ).toBe(luna!.endpoint);
    expect(
      getModelEndpoint(
        provider,
        "https://{resource%20name}.openai.azure.com/openai/v1/responses",
        luna!
      )
    ).toBe(luna!.endpoint);
    expect(
      getModelEndpoint(
        provider,
        "https://contoso.openai.azure.com/openai/v1/responses",
        luna!
      )
    ).toBe("https://contoso.openai.azure.com/openai/v1/responses");
    expect(
      getModelEndpoint(
        provider,
        "https://contoso.openai.azure.com/openai/deployments/legacy/chat/completions?api-version=2024-06-01",
        luna!
      )
    ).toBe("https://contoso.openai.azure.com/openai/v1/responses");
  });
});
