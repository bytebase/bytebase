import { fireEvent, render, screen } from "@testing-library/react";
import { expect, test, vi } from "vitest";
import { ExportGrantLookupStatus } from "./ExportGrantLookupStatus";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

test("pending lookup cannot be clicked; a failed lookup offers retry", () => {
  const retry = vi.fn();
  const { rerender } = render(<ExportGrantLookupStatus loading failed={false} onRetry={retry} />);
  expect(screen.getByRole("button").hasAttribute("disabled")).toBe(true);
  rerender(<ExportGrantLookupStatus loading={false} failed onRetry={retry} />);
  expect(screen.getByRole("alert").textContent).toContain("sql-editor.export-grant-lookup-failed");
  fireEvent.click(screen.getByRole("button", {name: "sql-editor.export-grant-lookup-retry"}));
  expect(retry).toHaveBeenCalledOnce();
  rerender(<ExportGrantLookupStatus loading={false} failed={false} onRetry={retry} />);
  expect(screen.queryByRole("button")).toBeNull();
});
