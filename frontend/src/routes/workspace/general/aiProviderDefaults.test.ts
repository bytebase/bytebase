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
      "gpt-5.6-terra"
    );
    expect(PROVIDER_DEFAULTS[AISetting_Provider.AZURE_OPENAI].model).toBe(
      "gpt-5.6-terra"
    );
    expect(PROVIDER_DEFAULTS[AISetting_Provider.GEMINI].model).toBe(
      "gemini-3.8-flash"
    );
    expect(PROVIDER_DEFAULTS[AISetting_Provider.CLAUDE].model).toBe(
      "claude-sonnet-5"
    );
  });

  test("includes current supported models", () => {
    expect(
      PROVIDER_MODELS[AISetting_Provider.OPEN_AI].map((model) => model.value)
    ).toEqual(
      expect.arrayContaining([
        "gpt-6-astra",
        "gpt-5.6-sol",
        "gpt-5.6-terra",
        "gpt-5.6-luna",
      ])
    );
    expect(
      PROVIDER_MODELS[AISetting_Provider.AZURE_OPENAI].map(
        (model) => model.value
      )
    ).toEqual(
      expect.arrayContaining([
        "gpt-6-astra",
        "gpt-5.6-sol",
        "gpt-5.6-terra",
        "gpt-5.6-luna",
      ])
    );
    expect(
      PROVIDER_MODELS[AISetting_Provider.GEMINI].map((model) => model.value)
    ).toEqual(
      expect.arrayContaining([
        "gemini-3.8-flash",
        "gemini-3.7-flash",
        "gemini-3.6-flash",
        "gemini-3.5-flash-lite",
      ])
    );
    expect(
      PROVIDER_MODELS[AISetting_Provider.CLAUDE].map((model) => model.value)
    ).toEqual(
      expect.arrayContaining([
        "claude-fable-5-1",
        "claude-opus-5",
        "claude-haiku-4-5",
      ])
    );
  });

  test("uses the Responses API for current OpenAI models", () => {
    const modelNames = [
      "gpt-6-astra",
      "gpt-5.6-sol",
      "gpt-5.6-terra",
      "gpt-5.6-luna",
      "gpt-5.5",
    ];
    for (const provider of [
      AISetting_Provider.OPEN_AI,
      AISetting_Provider.AZURE_OPENAI,
    ]) {
      for (const modelName of modelNames) {
        const model = PROVIDER_MODELS[provider].find(
          (model) => model.value === modelName
        );
        expect(model?.endpoint).toContain("/responses");
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

  test("uses the legacy Chat Completions API for GPT-4o", () => {
    for (const provider of [
      AISetting_Provider.OPEN_AI,
      AISetting_Provider.AZURE_OPENAI,
    ]) {
      const gpt4o = PROVIDER_MODELS[provider].find(
        (model) => model.value === "gpt-4o"
      );
      expect(gpt4o?.endpoint).toContain("/chat/completions");
    }
  });

  test("preserves the Azure resource origin when changing models", () => {
    const provider = AISetting_Provider.AZURE_OPENAI;
    const gpt4o = PROVIDER_MODELS[provider].find(
      (model) => model.value === "gpt-4o"
    );
    const gpt5 = PROVIDER_MODELS[provider].find(
      (model) => model.value === "gpt-5"
    );

    expect(gpt4o).toBeDefined();
    expect(gpt5).toBeDefined();
    expect(
      getModelEndpoint(provider, PROVIDER_DEFAULTS[provider].endpoint, gpt4o!)
    ).toBe(gpt4o!.endpoint);
    expect(
      getModelEndpoint(
        provider,
        "https://{resource%20name}.openai.azure.com/openai/v1/responses",
        gpt4o!
      )
    ).toBe(gpt4o!.endpoint);
    expect(
      getModelEndpoint(
        provider,
        "https://contoso.openai.azure.com/openai/v1/responses",
        gpt4o!
      )
    ).toBe("https://contoso.openai.azure.com/openai/v1/chat/completions");
    expect(
      getModelEndpoint(
        provider,
        "https://contoso.openai.azure.com/openai/deployments/legacy/chat/completions?api-version=2024-06-01",
        gpt5!
      )
    ).toBe("https://contoso.openai.azure.com/openai/v1/responses");
  });
});
