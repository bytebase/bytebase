import { create } from "@bufbuild/protobuf";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { AccessGrantSchema } from "@/types/proto-es/v1/access_grant_service_pb";
import { hashAccessGrantQuery } from "./accessGrantQueryHash";
import { useExportGrantBypass } from "./useExportGrantBypass";

const { searchMyAccessGrants, fetchIssueByName } = vi.hoisted(() => ({
  searchMyAccessGrants: vi.fn(), fetchIssueByName: vi.fn(),
}));
vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) => selector({searchMyAccessGrants, fetchIssueByName}),
}));
vi.mock("react-i18next", () => ({useTranslation: () => ({t: (key: string) => key})}));
vi.mock("@/components/RouterLink", () => ({RouterLink: () => null}));
const target = {database: "instances/i/databases/d", statement: "SELECT 1", schema: "", container: ""};
const grant = create(AccessGrantSchema, {name: "projects/p/accessGrants/g"});
const args = {enabled: true, project: "projects/p", targets: [target]};
beforeEach(() => vi.resetAllMocks());

test("long SQL is hashed and execution context remains exact", async () => {
  searchMyAccessGrants.mockResolvedValue({accessGrants: [grant]});
  const statement = "SELECT '" + "x".repeat(110_000) + "'";
  const {result} = renderHook(() => useExportGrantBypass({...args, targets: [{...target, statement, schema: "s", container: "c"}]}));
  await waitFor(() => expect(result.current.loading).toBe(false));
  expect(searchMyAccessGrants).toHaveBeenCalledWith({parent: "projects/p", pageSize: 1, filter: {queryHash: hashAccessGrantQuery(statement), schema: "s", container: "c", target: target.database, export: true, status: ["ACTIVE"]}});
  expect(JSON.stringify(searchMyAccessGrants.mock.calls)).not.toContain(statement);
});

test("failed batch targets stay separate from missing grants and can retry", async () => {
  searchMyAccessGrants.mockResolvedValueOnce({accessGrants: [grant]}).mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce({accessGrants: []});
  const targets = [target, {...target, database: "failed"}, {...target, database: "missing"}];
  const {result} = renderHook(() => useExportGrantBypass({...args, targets}));
  await waitFor(() => expect(result.current.loading).toBe(false));
  expect(result.current.matchedDatabases).toEqual([target.database]);
  expect(result.current.failedDatabases).toEqual(["failed"]);
  expect(result.current.unmatchedDatabases).toEqual(["missing"]);
  searchMyAccessGrants.mockResolvedValue({accessGrants: [grant]});
  act(() => result.current.retry());
  await waitFor(() => expect(result.current.matchedDatabases).toHaveLength(3));
  expect(result.current.failedDatabases).toEqual([]);
});

test("context changes hide prior grants immediately and ignore stale completions", async () => {
  let resolveOld!: (value: {accessGrants: typeof grant[]}) => void;
  searchMyAccessGrants.mockResolvedValueOnce({accessGrants: [grant]});
  const {result, rerender} = renderHook(({schema}) => useExportGrantBypass({...args, targets: [{...target, schema}]}), {initialProps: {schema: "one"}});
  await waitFor(() => expect(result.current.grantName).toBe(grant.name));
  searchMyAccessGrants.mockImplementationOnce(() => new Promise((resolve) => {resolveOld = resolve;}));
  rerender({schema: "two"});
  expect(result.current.grantName).toBe("");
  expect(result.current.unmatchedDatabases).toEqual([]);
  searchMyAccessGrants.mockResolvedValueOnce({accessGrants: []});
  rerender({schema: "three"});
  await waitFor(() => expect(result.current.loading).toBe(false));
  await act(async () => resolveOld({accessGrants: [grant]}));
  expect(result.current.grantName).toBe("");
  expect(result.current.unmatchedDatabases).toEqual([target.database]);
});
