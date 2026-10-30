import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { ProjectExportDestinationField } from "./ProjectExportDestinationField";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/FeatureBadge", () => ({ FeatureBadge: () => null }));

const savedUrl = "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents";

describe("ProjectExportDestinationField", () => {
  test("shows the saved URL and nothing resolved from it", () => {
    render(
      <ProjectExportDestinationField
        value={savedUrl}
        onChange={vi.fn()}
        disabled={false}
        canClear
      />
    );
    expect(
      screen.getByLabelText("export-data.destination.sharepoint")
    ).toHaveValue(savedUrl);
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  test("shows the server error beside the field", () => {
    render(
      <ProjectExportDestinationField
        value={savedUrl}
        onChange={vi.fn()}
        disabled={false}
        canClear
        error="does not exist or Bytebase cannot access it"
      />
    );
    expect(screen.getByRole("alert")).toHaveTextContent(
      "does not exist or Bytebase cannot access it"
    );
    expect(
      screen.getByLabelText("export-data.destination.sharepoint")
    ).toHaveAttribute("aria-invalid", "true");
  });

  test("clears the destination", () => {
    const onChange = vi.fn();
    render(
      <ProjectExportDestinationField
        value={savedUrl}
        onChange={onChange}
        disabled={false}
        canClear
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "common.clear" }));
    expect(onChange).toHaveBeenCalledWith("");
  });

  test("hides the clear button when there is nothing to clear", () => {
    render(
      <ProjectExportDestinationField
        value=""
        onChange={vi.fn()}
        disabled={false}
        canClear
      />
    );
    expect(
      screen.queryByRole("button", { name: "common.clear" })
    ).not.toBeInTheDocument();
  });

  test("can clear a destination after the license lapses", () => {
    const onChange = vi.fn();
    render(
      <ProjectExportDestinationField
        value={savedUrl}
        onChange={onChange}
        disabled
        canClear
      />
    );
    expect(
      screen.getByLabelText("export-data.destination.sharepoint")
    ).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "common.clear" }));
    expect(onChange).toHaveBeenCalledWith("");
  });
});
