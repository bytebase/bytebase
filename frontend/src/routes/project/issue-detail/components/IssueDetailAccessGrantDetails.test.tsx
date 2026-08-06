import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { IssueDetailAccessGrantDetails } from "./IssueDetailAccessGrantDetails";

const mocks = vi.hoisted(() => ({
  accessGrant: undefined as
    | {
        name: string;
        targets: string[];
        query: string;
        unmask: boolean;
        export: boolean;
        schema: string;
        container: string;
      }
    | undefined,
  issue: {
    name: "projects/proj/issues/123",
    accessGrant: "projects/proj/accessGrants/ag1",
  },
  project: { name: "projects/proj" },
  fetchAccessGrant: vi.fn(),
  searchMyAccessGrants: vi.fn(),
  getOrFetchDatabaseByName: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock("@/components/DatabaseTargetDisplay", () => ({
  DatabaseTargetDisplay: ({ target }: { target: string }) => (
    <span>{target}</span>
  ),
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => mocks.project,
}));

vi.mock("@/stores/app", () => {
  const state = () => ({
    projectsByName: {},
    fetchAccessGrant: mocks.fetchAccessGrant,
    searchMyAccessGrants: mocks.searchMyAccessGrants,
    getOrFetchDatabaseByName: mocks.getOrFetchDatabaseByName,
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

vi.mock("@/stores/modules/v1/common", () => ({
  projectNamePrefix: "projects/",
}));

vi.mock("@/types", () => ({
  isValidDatabaseName: (name: string) => name.includes("/databases/"),
}));

vi.mock("@/utils", () => ({
  extractProjectResourceName: () => "proj",
  hasProjectPermissionV2: () => true,
}));

vi.mock("@/utils/accessGrant", () => ({
  getAccessGrantExpirationText: () => ({ type: "never" }),
}));

vi.mock("../context/IssueDetailContext", () => ({
  useIssueDetailContext: () => ({ issue: mocks.issue, projectId: "proj" }),
}));

beforeEach(() => {
  vi.clearAllMocks();
  mocks.accessGrant = {
    name: "projects/proj/accessGrants/ag1",
    targets: ["instances/inst/databases/db"],
    query: "SELECT * FROM orders",
    unmask: true,
    export: true,
    schema: "APP",
    container: "orders",
  };
  mocks.fetchAccessGrant.mockImplementation(async () => mocks.accessGrant);
  mocks.searchMyAccessGrants.mockResolvedValue({ accessGrants: [] });
});

describe("IssueDetailAccessGrantDetails", () => {
  test("renders schema and CosmosDB container for approvers", async () => {
    render(<IssueDetailAccessGrantDetails />);

    expect(await screen.findByText("APP")).toBeInTheDocument();
    expect(screen.getByText("orders")).toBeInTheDocument();
    expect(screen.getByText("common.schema")).toBeInTheDocument();
    expect(screen.getByText("issue.access-grant.container")).toBeInTheDocument();
  });
});
