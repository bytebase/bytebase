import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { AccountTypeBadge } from "./AccountTypeBadge";

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

describe("AccountTypeBadge", () => {
  test("marks nothing for a person", () => {
    const { container } = render(<AccountTypeBadge email="alex@example.com" />);
    expect(container).toBeEmptyDOMElement();
  });

  test.each([
    ["deploy@service.bytebase.com", "settings.members.service-account"],
    ["ci@orders.workload.bytebase.com", "settings.members.workload-identity"],
  ])("names the account type of %s", (email, label) => {
    render(<AccountTypeBadge email={email} />);
    expect(screen.getByText(label)).toBeInTheDocument();
  });

  // The name beside it identifies the principal; the mark is secondary
  // metadata, and accent is reserved for primary action and selected state.
  test.each([
    false,
    true,
  ])("never draws the mark in the accent role (compact: %s)", (compact) => {
    render(
      <AccountTypeBadge compact={compact} email="deploy@service.bytebase.com" />
    );
    expect(
      screen.getByText("settings.members.service-account").className
    ).not.toMatch(/accent/);
  });
});
