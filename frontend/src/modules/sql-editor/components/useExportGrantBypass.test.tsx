import { create } from "@bufbuild/protobuf";
import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { AccessGrantSchema } from "@/types/proto-es/v1/access_grant_service_pb";
import { hashAccessGrantQuery } from "./accessGrantQueryHash";
import * as queryHashUtils from "./accessGrantQueryHash";
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
afterEach(() => vi.restoreAllMocks());

test("large batches keep render keys compact and reuse hashes across renders", async () => {
  const hash = vi.spyOn(queryHashUtils, "hashAccessGrantQuery");
  const stringify = JSON.stringify;
  let largestSerialization = 0;
  vi.spyOn(JSON, "stringify").mockImplementation((value, replacer, space) => {
    const serialized = stringify(value, replacer as never, space);
    largestSerialization = Math.max(largestSerialization, serialized?.length ?? 0);
    return serialized;
  });
  searchMyAccessGrants.mockResolvedValue({ accessGrants: [grant] });
  const statement = "SELECT '" + "x".repeat(110_000) + "'";
  const { result, rerender } = renderHook(({ sql }) => useExportGrantBypass({
    ...args,
    targets: Array.from({ length: 200 }, (_, i) => ({ ...target, database: `db${i}`, statement: sql })),
  }), { initialProps: { sql: statement } });
  await waitFor(() => expect(result.current.matchedDatabases).toHaveLength(200));
  expect(hash).toHaveBeenCalledTimes(1);
  expect(searchMyAccessGrants).toHaveBeenCalledTimes(200);

  rerender({ sql: statement });
  expect(hash).toHaveBeenCalledTimes(1);
  expect(searchMyAccessGrants).toHaveBeenCalledTimes(200);
  expect(largestSerialization).toBeLessThan(50_000);

  rerender({ sql: statement + " AS changed" });
  expect(result.current.grantName).toBe("");
  await waitFor(() => expect(result.current.matchedDatabases).toHaveLength(200));
  expect(hash).toHaveBeenCalledTimes(2);
  expect(searchMyAccessGrants).toHaveBeenCalledTimes(400);
});

test("hash failures remain retryable lookup errors", async () => {
  const hash = vi.spyOn(queryHashUtils, "hashAccessGrantQuery").mockImplementation(() => {
    throw new Error("invalid UTF-8");
  });
  const { result } = renderHook(() => useExportGrantBypass(args));
  await waitFor(() => expect(result.current.failedDatabases).toEqual([target.database]));
  expect(result.current.unmatchedDatabases).toEqual([]);
  expect(searchMyAccessGrants).not.toHaveBeenCalled();
  hash.mockRestore();
  searchMyAccessGrants.mockResolvedValue({ accessGrants: [grant] });
  act(() => result.current.retry());
  await waitFor(() => expect(result.current.grantName).toBe(grant.name));
});

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
