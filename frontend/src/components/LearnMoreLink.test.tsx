import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { LearnMoreLink } from "./LearnMoreLink";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) =>
      ({
        "common.learn-more": "Learn more",
      })[key] ?? key,
  }),
}));

describe("LearnMoreLink", () => {
  test("renders an underlined link", () => {
    render(<LearnMoreLink href="https://docs.bytebase.com" />);

    expect(screen.getByRole("link", { name: /learn more/i })).toHaveClass(
      "underline"
    );
  });
});
