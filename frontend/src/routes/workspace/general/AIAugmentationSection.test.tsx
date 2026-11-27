import { create } from "@bufbuild/protobuf";
import { createRef, type ReactNode } from "react";
import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type AISetting,
  AISetting_Provider,
  AISettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import type { SectionHandle } from "./useSettingSection";

const mocks = vi.hoisted(() => ({
  getOrFetchSettingByName: vi.fn(),
  upsertSetting: vi.fn(),
  refreshServerInfo: vi.fn(),
  isSaaSMode: false,
  aiEnabled: false as boolean | undefined,
  aiSetting: undefined as AISetting | undefined,
  canEdit: true,
  canRead: true,
  settingsByName: {},
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/hooks/useAppState", () => ({
  useServerState: () => ({ isSaaSMode: mocks.isSaaSMode }),
}));
vi.mock("@/lib/productIntro", () => ({
  AI_ASSISTANT_PRODUCT_INTRO: "ai-assistant",
  useProductIntro: vi.fn(),
}));
vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: (permission: string) =>
    permission === "bb.settings.get" ? mocks.canRead : mocks.canEdit,
}));
vi.mock("@/stores/app", () => {
  const state = {
    get settingsByName() {
      return mocks.settingsByName;
    },
    getSettingByName: () =>
      mocks.aiSetting
        ? { value: { value: { case: "ai", value: mocks.aiSetting } } }
        : undefined,
    getOrFetchSettingByName: mocks.getOrFetchSettingByName,
    upsertSetting: mocks.upsertSetting,
    refreshServerInfo: mocks.refreshServerInfo,
    get serverInfo() {
      return { aiEnabled: mocks.aiEnabled };
    },
  };
  return {
    useAppStore: Object.assign(
      (selector: (value: typeof state) => unknown) => selector(state),
      { getState: () => state }
    ),
  };
});
vi.mock("@/components/ComponentPermissionGuard", () => ({
  ComponentPermissionGuard: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/components/LearnMoreLink", () => ({
  LearnMoreLink: () => null,
}));
vi.mock("@/components/ui/alert", () => ({
  Alert: ({ description }: { description: ReactNode }) => <div>{description}</div>,
}));
vi.mock("@/components/ui/form", () => ({
  FormLabel: (props: React.LabelHTMLAttributes<HTMLLabelElement>) => <label {...props} />,
  FormField: ({ children, title }: { children?: ReactNode; title: ReactNode }) => (
    <div>{title}{children}</div>
  ),
  FormFieldGroup: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  FormSection: ({ children, id, title }: { children: ReactNode; id?: string; title: ReactNode }) => (
    <section id={id}><div role="heading" aria-level={2}>{title}</div>{children}</section>
  ),
}));
vi.mock("@/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
}));
vi.mock("@/components/ui/switch", () => ({
  Switch: ({ checked, disabled, onCheckedChange }: { checked: boolean; disabled?: boolean; onCheckedChange: (checked: boolean) => void }) => (
    <input
      aria-label="enable-ai"
      type="checkbox"
      checked={checked}
      disabled={disabled}
      onChange={(event) => onCheckedChange(event.target.checked)}
    />
  ),
}));

import { AIAugmentationSection } from "./AIAugmentationSection";

describe("AIAugmentationSection", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.isSaaSMode = false;
    mocks.aiEnabled = false;
    mocks.aiSetting = undefined;
    mocks.canEdit = true;
    mocks.canRead = true;
    mocks.settingsByName = {};
    mocks.upsertSetting.mockResolvedValue(undefined);
    mocks.refreshServerInfo.mockResolvedValue(undefined);
  });

  test("shows the saved enabled state beside the section title, not the draft toggle state", () => {
    mocks.aiSetting = create(AISettingSchema, { enabled: true });
    const { rerender } = render(<AIAugmentationSection title="AI" onDirtyChange={vi.fn()} />);

    expect(within(screen.getByRole("heading", { name: /AI/ })).getByText("common.enabled")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: "enable-ai" }));
    expect(screen.getByText("common.enabled")).toBeInTheDocument();

    mocks.aiSetting = create(AISettingSchema, { enabled: false });
    mocks.settingsByName = {};
    rerender(<AIAugmentationSection title="AI" onDirtyChange={vi.fn()} />);
    expect(screen.queryByText("common.enabled")).not.toBeInTheDocument();
  });

  test.each([undefined, false])("does not show Enabled while the saved state is %s", (enabled) => {
    mocks.aiSetting = enabled === undefined ? undefined : create(AISettingSchema, { enabled });
    render(<AIAugmentationSection title="AI" onDirtyChange={vi.fn()} />);

    expect(screen.queryByText("common.enabled")).not.toBeInTheDocument();
  });

  test("does not reveal saved status to a user without permission to read AI settings", () => {
    mocks.aiSetting = create(AISettingSchema, { enabled: true });
    mocks.canRead = false;
    render(<AIAugmentationSection title="AI" onDirtyChange={vi.fn()} />);

    expect(screen.queryByText("common.enabled")).not.toBeInTheDocument();
  });

  test("uses a display-only mask for the existing key and never sends it when saving another field", async () => {
    mocks.aiSetting = create(AISettingSchema, {
      enabled: true,
      provider: AISetting_Provider.GEMINI,
      endpoint: "https://api.example.com",
      model: "example-model",
    });
    const ref = createRef<SectionHandle>();
    render(<AIAugmentationSection ref={ref} title="AI" onDirtyChange={vi.fn()} />);

    const input = screen.getByLabelText("settings.general.workspace.ai-assistant.api-key.self");
    expect(input).toHaveAttribute("placeholder", "****************");
    expect(input).toHaveAccessibleDescription("settings.general.workspace.ai-assistant.api-key.stored-description");
    expect(input).toHaveValue("");
    expect(ref.current?.isDirty()).toBe(false);

    fireEvent.change(input, { target: { value: "replacement-key" } });
    expect(input).not.toHaveAttribute("aria-description");
    fireEvent.change(input, { target: { value: "" } });

    fireEvent.change(screen.getByDisplayValue("example-model"), { target: { value: "next-model" } });
    await act(async () => ref.current?.update());

    const request = mocks.upsertSetting.mock.calls[0][0];
    expect(request.updateMask.paths).toEqual(["value.ai.model"]);
    expect(request.value.value.value.apiKey).toBe("");
  });

  test("removes the stored-key mask when the endpoint changes and restores it on revert", () => {
    mocks.aiSetting = create(AISettingSchema, {
      enabled: true,
      provider: AISetting_Provider.GEMINI,
      endpoint: "https://api.example.com",
      model: "example-model",
    });
    const ref = createRef<SectionHandle>();
    render(<AIAugmentationSection ref={ref} title="AI" onDirtyChange={vi.fn()} />);

    fireEvent.change(screen.getByDisplayValue("https://api.example.com"), { target: { value: "https://custom.example.com" } });
    expect(screen.getByLabelText("settings.general.workspace.ai-assistant.api-key.self")).toHaveAttribute("placeholder", "common.sensitive-placeholder");
    act(() => ref.current?.revert());
    expect(screen.getByLabelText("settings.general.workspace.ai-assistant.api-key.self")).toHaveAttribute("placeholder", "****************");
  });

  test.each([false, true])(
    "allows toggling Cloud AI when enabled is %s without changing credentials",
    async (enabled) => {
      mocks.isSaaSMode = true;
      mocks.aiSetting = create(AISettingSchema, {
        enabled,
        provider: AISetting_Provider.GEMINI,
        endpoint: "https://api.example.com",
        model: "example-model",
      });
      const ref = createRef<SectionHandle>();
      render(<AIAugmentationSection ref={ref} title="AI" onDirtyChange={vi.fn()} />);

      const toggle = screen.getByRole("checkbox", { name: "enable-ai" });
      expect(toggle).toBeEnabled();
      expect(ref.current?.isDirty()).toBe(false);
      fireEvent.click(toggle);
      expect(ref.current?.isDirty()).toBe(true);
      await act(async () => ref.current?.update());

      const request = mocks.upsertSetting.mock.calls[0][0];
      expect(request.updateMask.paths).toEqual(["value.ai.enabled"]);
      expect(request.value.value.value.enabled).toBe(!enabled);
    }
  );

  test("allows Cloud users to change provider and credentials", async () => {
    mocks.isSaaSMode = true;
    mocks.aiSetting = create(AISettingSchema, {
      enabled: true,
      provider: AISetting_Provider.GEMINI,
      endpoint: "https://api.example.com",
      model: "example-model",
    });
    const ref = createRef<SectionHandle>();
    render(<AIAugmentationSection ref={ref} title="AI" onDirtyChange={vi.fn()} />);

    fireEvent.click(screen.getByRole("combobox", { name: "settings.general.workspace.ai-assistant.provider.self" }));
    const provider = await screen.findByRole("option", { name: "settings.general.workspace.ai-assistant.provider.claude" });
    fireEvent.pointerDown(provider, { pointerType: "mouse" });
    fireEvent.click(provider);
    expect(screen.getByLabelText("settings.general.workspace.ai-assistant.api-key.self")).toHaveAttribute("placeholder", "common.sensitive-placeholder");
    fireEvent.change(screen.getByLabelText("settings.general.workspace.ai-assistant.api-key.self"), {
      target: { value: "user-api-key" },
    });
    await act(async () => ref.current?.update());

    const request = mocks.upsertSetting.mock.calls[0][0];
    expect(request.value.value.value.provider).toBe(AISetting_Provider.CLAUDE);
    expect(request.value.value.value.apiKey).toBe("user-api-key");
    expect(request.updateMask.paths).toEqual([
      "value.ai.provider", "value.ai.endpoint", "value.ai.model", "value.ai.api_key",
    ]);
  });

  test.each([false, true])("keeps settings read-only without permission in SaaS mode %s", (isSaaSMode) => {
    mocks.isSaaSMode = isSaaSMode;
    mocks.canEdit = false;
    mocks.aiSetting = create(AISettingSchema, {
      enabled: true,
      provider: AISetting_Provider.GEMINI,
      endpoint: "https://api.example.com",
      model: "example-model",
    });
    render(<AIAugmentationSection title="AI" onDirtyChange={vi.fn()} />);

    expect(screen.getByRole("checkbox", { name: "enable-ai" })).toBeDisabled();
    expect(screen.getByLabelText("settings.general.workspace.ai-assistant.api-key.self")).toBeDisabled();
    for (const control of screen.getAllByRole("combobox")) expect(control).toBeDisabled();
  });

  test("explicitly updates the API key when changing the endpoint", async () => {
    mocks.isSaaSMode = true;
    mocks.aiSetting = create(AISettingSchema, {
      enabled: true,
      provider: AISetting_Provider.GEMINI,
      endpoint: "https://api.example.com",
      model: "example-model",
    });
    const ref = createRef<SectionHandle>();
    render(<AIAugmentationSection ref={ref} title="AI" onDirtyChange={vi.fn()} />);
    fireEvent.change(screen.getByDisplayValue("https://api.example.com"), {
      target: { value: "https://custom.example.com" },
    });
    await act(async () => ref.current?.update());

    const request = mocks.upsertSetting.mock.calls[0][0];
    expect(request.updateMask.paths).toEqual(["value.ai.endpoint", "value.ai.api_key"]);
    expect(request.value.value.value.apiKey).toBe("");
  });

  test("exposes the AI settings section as a hash target without scrolling itself", () => {
    const originalUrl = window.location.href;
    const originalScrollIntoView = HTMLElement.prototype.scrollIntoView;
    const scrollIntoView = vi.fn();
    HTMLElement.prototype.scrollIntoView = scrollIntoView;
    window.history.replaceState(null, "", "#ai-assistant");

    try {
      const { container } = render(
        <AIAugmentationSection title="AI" onDirtyChange={vi.fn()} />
      );
      expect(container.querySelector("#ai-assistant")).toBeTruthy();
      expect(scrollIntoView).not.toHaveBeenCalled();
    } finally {
      window.history.replaceState(null, "", originalUrl);
      HTMLElement.prototype.scrollIntoView = originalScrollIntoView;
    }
  });

  test("saves AI without depending on a second actuator request", async () => {
    mocks.refreshServerInfo.mockRejectedValue(new Error("refresh failed"));
    const ref = createRef<SectionHandle>();
    render(
      <AIAugmentationSection ref={ref} title="AI" onDirtyChange={vi.fn()} />
    );

    fireEvent.click(screen.getByRole("checkbox", { name: "enable-ai" }));
    await act(async () => ref.current?.update());

    expect(mocks.upsertSetting).toHaveBeenCalledOnce();
    expect(mocks.refreshServerInfo).not.toHaveBeenCalled();
  });

  test("includes the selected provider when creating the AI setting", async () => {
    const ref = createRef<SectionHandle>();
    render(
      <AIAugmentationSection ref={ref} title="AI" onDirtyChange={vi.fn()} />
    );

    fireEvent.click(screen.getByRole("checkbox", { name: "enable-ai" }));
    await act(async () => ref.current?.update());

    const request = mocks.upsertSetting.mock.calls[0][0];
    expect(request.updateMask.paths).toContain("value.ai.provider");
  });

  test.each([
    ["OpenAI", AISetting_Provider.OPEN_AI],
    ["Azure OpenAI", AISetting_Provider.AZURE_OPENAI],
    ["Gemini", AISetting_Provider.GEMINI],
    ["Claude", AISetting_Provider.CLAUDE],
  ])("protects the %s API key while preserving updates", async (_name, provider) => {
    mocks.aiSetting = create(AISettingSchema, {
      enabled: true,
      provider,
      endpoint: "https://api.example.com",
      model: "example-model",
    });
    const ref = createRef<SectionHandle>();
    render(<AIAugmentationSection ref={ref} title="AI" onDirtyChange={vi.fn()} />);

    const input = screen.getByLabelText(
      "settings.general.workspace.ai-assistant.api-key.self"
    ) as HTMLInputElement;
    expect(input.type).toBe("password");
    expect(input.autocomplete).toBe("new-password");
    expect(input.closest("[data-agent-sensitive]")).not.toBeNull();
    expect(screen.getByLabelText(
      "settings.general.workspace.ai-assistant.api-key.self"
    )).toBe(input);

    fireEvent.change(input, { target: { value: "example-api-key" } });
    expect(input.type).toBe("password");
    fireEvent.click(screen.getByRole("button", {
      name: "common.toggle-password-visibility",
    }));
    expect(input.type).toBe("text");
    expect(input.value).toBe("example-api-key");

    await act(async () => ref.current?.update());
    const request = mocks.upsertSetting.mock.calls[0][0];
    expect(request.value.value.value.apiKey).toBe("example-api-key");
    expect(request.value.value.value.provider).toBe(provider);
    expect(request.updateMask.paths).toContain("value.ai.api_key");
  });

  test.each(["revert", "update"] as const)(
    "masks the next API key after %s resets a revealed draft",
    async (action) => {
      mocks.aiSetting = create(AISettingSchema, {
        enabled: true,
        provider: AISetting_Provider.GEMINI,
        endpoint: "https://api.example.com",
        model: "example-model",
      });
      const ref = createRef<SectionHandle>();
      render(<AIAugmentationSection ref={ref} title="AI" onDirtyChange={vi.fn()} />);
      const label = "settings.general.workspace.ai-assistant.api-key.self";
      fireEvent.change(screen.getByLabelText(label), {
        target: { value: "example-api-key" },
      });
      fireEvent.click(screen.getByRole("button", {
        name: "common.toggle-password-visibility",
      }));
      expect(screen.getByLabelText(label)).toHaveAttribute("type", "text");

      await act(async () => { await ref.current?.[action](); });

      fireEvent.change(screen.getByLabelText(label), {
        target: { value: "next-example-api-key" },
      });
      expect(screen.getByLabelText(label)).toHaveAttribute("type", "password");
    }
  );
});
