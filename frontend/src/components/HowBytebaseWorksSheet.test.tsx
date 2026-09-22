import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import guideContent from "@/locales/guides/how-bytebase-works.en-US.md?raw";
import { HowBytebaseWorksSheet } from "./HowBytebaseWorksSheet";

const mocks = vi.hoisted(() => ({
  captureMetric: vi.fn(),
  resolvedLanguage: "en-US",
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { resolvedLanguage: mocks.resolvedLanguage },
  }),
}));

vi.mock("@/app/analytics/provider", () => ({
  behaviorAnalytics: {
    captureMetric: mocks.captureMetric,
  },
}));

describe("HowBytebaseWorksSheet", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.resolvedLanguage = "en-US";
  });

  test("keeps the product model concise in Markdown", () => {
    expect(guideContent).toContain(
      "Bytebase sits between your team and your databases"
    );
    expect(guideContent).toContain("- **Workspace:**");
    expect(guideContent).toContain("- **Project:**");
    expect(guideContent).toContain("- **Instance:**");
    expect(guideContent).toContain("- **Database:**");
    expect(guideContent).toContain("- **Query data:**");
    expect(guideContent).toContain("- **Change data:**");
    expect(guideContent).not.toContain("## ");
  });

  test("renders the documentation-aligned explanation without an abstract diagram", async () => {
    render(<HowBytebaseWorksSheet open onOpenChange={vi.fn()} />);

    expect(
      await screen.findByText("workspace-setup-guide.product-model")
    ).toBeInTheDocument();
    expect(
      screen.getByText(/sits between your team and your databases/i)
    ).toBeInTheDocument();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();

    const learnMoreLink = screen.getByRole("link", {
      name: "Learn more about how Bytebase organizes resources",
    });
    expect(learnMoreLink).toHaveAttribute(
      "href",
      "https://docs.bytebase.com/onboarding/organize-resources"
    );
    expect(learnMoreLink).toHaveAttribute("target", "_blank");
  });

  test("does not record guide analytics for the learn more action", async () => {
    render(<HowBytebaseWorksSheet open onOpenChange={vi.fn()} />);

    fireEvent.click(
      await screen.findByRole("link", {
        name: "Learn more about how Bytebase organizes resources",
      })
    );
    expect(mocks.captureMetric).not.toHaveBeenCalled();
  });

  test.each([
    {
      locale: "zh-CN",
      intro: "Bytebase 位于你的团队和数据库之间",
      link: "进一步了解 Bytebase 如何组织资源",
    },
    {
      locale: "es-ES",
      intro: "Bytebase se sitúa entre tu equipo y tus bases de datos",
      link: "Más información sobre cómo Bytebase organiza los recursos",
    },
    {
      locale: "ja-JP",
      intro: "Bytebase はチームとデータベースの間に位置し",
      link: "Bytebase のリソース構成について詳しく見る",
    },
    {
      locale: "vi-VN",
      intro: "Bytebase nằm giữa nhóm của bạn và các cơ sở dữ liệu",
      link: "Tìm hiểu thêm về cách Bytebase tổ chức tài nguyên",
    },
  ])("renders the $locale guide", async ({ locale, intro, link }) => {
    mocks.resolvedLanguage = locale;

    render(<HowBytebaseWorksSheet open onOpenChange={vi.fn()} />);

    expect(await screen.findByText(new RegExp(intro))).toBeInTheDocument();
    expect(screen.getByRole("link", { name: link })).toHaveAttribute(
      "href",
      "https://docs.bytebase.com/onboarding/organize-resources"
    );
  });

  test("does not render without content for the resolved locale", () => {
    mocks.resolvedLanguage = "fr-FR";

    render(<HowBytebaseWorksSheet open onOpenChange={vi.fn()} />);

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
