import { createRef, type ReactNode } from "react";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SectionHandle } from "./useSettingSection";

const mocks = vi.hoisted(() => ({
  getOrFetchSettingByName: vi.fn(),
  upsertSetting: vi.fn(),
  refreshServerInfo: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/hooks/useAppState", () => ({
  useServerState: () => ({ isSaaSMode: false }),
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
  FormSection: ({ children }: { children: ReactNode }) => <section>{children}</section>,
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
    mocks.upsertSetting.mockResolvedValue(undefined);
    mocks.refreshServerInfo.mockResolvedValue(undefined);
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
