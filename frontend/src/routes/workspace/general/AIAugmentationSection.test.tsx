import { createRef, type ReactNode } from "react";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SectionHandle } from "./useSettingSection";

const mocks = vi.hoisted(() => ({
  getOrFetchSettingByName: vi.fn(),
  upsertSetting: vi.fn(),
  refreshServerInfo: vi.fn(),
  isSaaSMode: false,
  aiEnabled: false as boolean | undefined,
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
  hasWorkspacePermissionV2: () => true,
}));
vi.mock("@/stores/app", () => {
  const state = {
    settingsByName: {},
    getSettingByName: () => undefined,
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
  FormField: ({ children, title }: { children?: ReactNode; title: ReactNode }) => (
    <div>{title}{children}</div>
  ),
  FormFieldGroup: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  FormSection: ({ children, id }: { children: ReactNode; id?: string }) => (
    <section id={id}>{children}</section>
  ),
}));
vi.mock("@/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
}));
vi.mock("@/components/ui/select", () => ({
  Select: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SelectContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SelectItem: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SelectTrigger: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SelectValue: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));
vi.mock("@/components/ui/switch", () => ({
  Switch: ({ checked, onCheckedChange }: { checked: boolean; onCheckedChange: (checked: boolean) => void }) => (
    <input
      aria-label="enable-ai"
      type="checkbox"
      checked={checked}
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
    mocks.upsertSetting.mockResolvedValue(undefined);
    mocks.refreshServerInfo.mockResolvedValue(undefined);
  });

  test.each([
    [true, "available-in-saas"],
    [false, "unavailable-in-saas"],
  ])(
    "shows actual SaaS AI availability when aiEnabled is %s",
    (aiEnabled, message) => {
      mocks.isSaaSMode = true;
      mocks.aiEnabled = aiEnabled;
      render(<AIAugmentationSection title="AI" onDirtyChange={vi.fn()} />);

      expect(
        screen.getByText(`settings.general.workspace.ai-assistant.${message}`)
      ).toBeTruthy();
      expect(
        screen.queryByText(
          "settings.general.workspace.ai-assistant.enabled-in-saas"
        )
      ).toBeNull();
      expect(screen.queryByRole("checkbox", { name: "enable-ai" })).toBeNull();
    }
  );

  test("does not report AI unavailable before server info loads", () => {
    mocks.isSaaSMode = true;
    mocks.aiEnabled = undefined;
    render(<AIAugmentationSection title="AI" onDirtyChange={vi.fn()} />);

    expect(
      screen.queryByText(
        "settings.general.workspace.ai-assistant.unavailable-in-saas"
      )
    ).toBeNull();
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
});
