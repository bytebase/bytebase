import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { SharePointLocationSchema } from "@/types/proto-es/v1/project_service_pb";
import { ProjectExportDestinationField } from "./ProjectExportDestinationField";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/FeatureBadge", () => ({ FeatureBadge: () => null }));

const savedUrl = "https://contoso.sharepoint.com/sites/Finance/Shared%20Documents";
const savedLocation = create(SharePointLocationSchema, {
  siteName: "Finance",
  libraryName: "Documents",
  folderPath: "Exports",
  webUrl: savedUrl,
});

describe("ProjectExportDestinationField", () => {
  test("labels the saved destination as found, not verified", () => {
    render(
      <ProjectExportDestinationField
        value={savedUrl}
        onChange={vi.fn()}
        disabled={false}
        canClear
        savedUrl={savedUrl}
        savedLocation={savedLocation}
      />
    );
    expect(
      screen.getByText("project.settings.export-destination.found")
    ).toBeInTheDocument();
    expect(
      screen.getByText("project.settings.export-destination.found-description")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /Finance \/ Documents \/ Exports/ })
    ).toHaveAttribute("href", savedUrl);
  });

  test("hides the saved location once the URL is edited", () => {
    render(
      <ProjectExportDestinationField
        value="https://contoso.sharepoint.com/sites/Other"
        onChange={vi.fn()}
        disabled={false}
        canClear
        savedUrl={savedUrl}
        savedLocation={savedLocation}
      />
    );
    expect(
      screen.queryByText("project.settings.export-destination.found")
    ).not.toBeInTheDocument();
  });

  test("shows the server error beside the field", () => {
    render(
      <ProjectExportDestinationField
        value={savedUrl}
        onChange={vi.fn()}
        disabled={false}
        canClear
        savedUrl={savedUrl}
        savedLocation={savedLocation}
        error="does not exist or Bytebase cannot access it"
      />
    );
    expect(screen.getByRole("alert")).toHaveTextContent(
      "does not exist or Bytebase cannot access it"
    );
    expect(
      screen.getByLabelText("project.settings.export-destination.self")
    ).toHaveAttribute("aria-invalid", "true");
    expect(
      screen.queryByText("project.settings.export-destination.found")
    ).not.toBeInTheDocument();
  });

  test("clears the destination", () => {
    const onChange = vi.fn();
    render(
      <ProjectExportDestinationField
        value={savedUrl}
        onChange={onChange}
        disabled={false}
        canClear
        savedUrl={savedUrl}
        savedLocation={savedLocation}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "common.clear" }));
    expect(onChange).toHaveBeenCalledWith("");
  });

  test("can clear a destination after the license lapses", () => {
    const onChange = vi.fn();
    render(
      <ProjectExportDestinationField
        value={savedUrl}
        onChange={onChange}
        disabled
        canClear
        savedUrl={savedUrl}
        savedLocation={savedLocation}
      />
    );
    expect(
      screen.getByLabelText("project.settings.export-destination.self")
    ).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "common.clear" }));
    expect(onChange).toHaveBeenCalledWith("");
  });
});
