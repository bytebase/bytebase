import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";
import { Button } from "@/components/ui/button";
import { AgentErrorDetails } from "./AgentErrorDetails";

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

afterEach(cleanup);

test("keeps multiple actions in a dedicated bottom row below diagnostics", () => {
  render(
    <AgentErrorDetails
      error="Connection failed"
      actions={<><Button size="sm">Retry</Button><Button size="sm">End run</Button></>}
    />
  );
  const details = screen.getByRole("button", { name: "error-page.error-details" });
  const retry = screen.getByRole("button", { name: "Retry" });
  const footer = retry.closest("[data-agent-error-actions]");
  expect(footer).toContainElement(retry);
  expect(footer).toContainElement(screen.getByRole("button", { name: "End run" }));
  expect(footer).not.toContainElement(details);
  expect(footer).toHaveClass("flex-wrap", "justify-end", "w-full");
  expect(details.compareDocumentPosition(retry) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  expect(screen.queryByText("Connection failed")).toBeNull();

  fireEvent.click(details);

  const diagnostic = screen.getByText("Connection failed");
  expect(details).toHaveAttribute("aria-expanded", "true");
  expect(footer).not.toContainElement(diagnostic);
  expect(footer!.parentElement).toContainElement(diagnostic);
  expect(diagnostic.compareDocumentPosition(footer!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
});

test("keeps recovery actionable without an empty diagnostic disclosure", () => {
  const onRetry = vi.fn();
  render(<AgentErrorDetails actions={<Button onClick={onRetry}>Retry</Button>} />);
  expect(screen.queryByRole("button", { name: "error-page.error-details" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Retry" }));
  expect(onRetry).toHaveBeenCalledOnce();
});

test("preserves standalone diagnostic disclosure", () => {
  render(<AgentErrorDetails error="Connection failed" />);
  const details = screen.getByRole("button", { name: "error-page.error-details" });
  fireEvent.click(details);
  expect(screen.getByText("Connection failed")).toBeVisible();
  fireEvent.click(details);
  expect(screen.queryByText("Connection failed")).toBeNull();
});

test("renders nothing without diagnostics or actions", () => {
  const { container } = render(<AgentErrorDetails />);
  expect(container).toBeEmptyDOMElement();
});
