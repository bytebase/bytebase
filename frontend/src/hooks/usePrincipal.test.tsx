import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { usePrincipal } from "./usePrincipal";

// The store's getters as they behave: a person is undefined until fetched; a
// service account or workload identity is a placeholder named from its email.
const store = vi.hoisted(() => {
  const state = {
    usersByName: {} as Record<string, { title: string; email: string }>,
    serviceAccountsByName: {} as Record<string, { title: string }>,
    workloadIdentitiesByName: {} as Record<string, { title: string }>,
    getUserByIdentifier: (identifier: string) => state.usersByName[identifier],
    getServiceAccount: (name: string) =>
      state.serviceAccountsByName[name] ?? {
        title: name.slice("serviceAccounts/".length).split("@")[0],
      },
    getWorkloadIdentity: (name: string) =>
      state.workloadIdentitiesByName[name] ?? {
        title: name.slice("workloadIdentities/".length).split("@")[0],
      },
    getOrFetchUserByIdentifier: vi.fn(),
    fetchServiceAccount: vi.fn(),
    fetchWorkloadIdentity: vi.fn(),
  };
  return state;
});

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (s: unknown) => unknown) => selector(store),
    { getState: () => store }
  ),
}));

beforeEach(() => {
  store.usersByName = {};
  store.serviceAccountsByName = {};
  store.workloadIdentitiesByName = {};
  store.getOrFetchUserByIdentifier.mockReset();
  store.fetchServiceAccount.mockReset();
  store.fetchWorkloadIdentity.mockReset();
});

describe("usePrincipal", () => {
  test("a person is read from the user cache and fetched through the user service", () => {
    store.usersByName["users/alex@example.com"] = {
      title: "Alex Kim",
      email: "alex@example.com",
    };
    const { result } = renderHook(() => usePrincipal("users/alex@example.com"));
    expect(result.current).toEqual({ email: "alex@example.com", title: "Alex Kim" });
    expect(store.getOrFetchUserByIdentifier).toHaveBeenCalledWith({
      identifier: "users/alex@example.com",
    });
    expect(store.fetchServiceAccount).not.toHaveBeenCalled();
  });

  test("a person not yet fetched has no title", () => {
    const { result } = renderHook(() => usePrincipal("users/new@example.com"));
    expect(result.current).toEqual({ email: "new@example.com", title: "" });
  });

  test("a service account is fetched from its own service and named from its email until its record arrives", () => {
    const { result, rerender } = renderHook(() =>
      usePrincipal("users/deploy@service.bytebase.com")
    );
    expect(result.current).toEqual({
      email: "deploy@service.bytebase.com",
      title: "deploy",
    });
    expect(store.fetchServiceAccount).toHaveBeenCalledWith(
      "serviceAccounts/deploy@service.bytebase.com",
      true
    );
    expect(store.getOrFetchUserByIdentifier).not.toHaveBeenCalled();

    store.serviceAccountsByName["serviceAccounts/deploy@service.bytebase.com"] = {
      title: "Deploy bot",
    };
    rerender();
    expect(result.current?.title).toBe("Deploy bot");
  });

  test("a workload identity likewise", () => {
    const { result } = renderHook(() =>
      usePrincipal("users/ci@workload.bytebase.com")
    );
    expect(result.current?.title).toBe("ci");
    expect(store.fetchWorkloadIdentity).toHaveBeenCalledWith(
      "workloadIdentities/ci@workload.bytebase.com",
      true
    );
    expect(store.getOrFetchUserByIdentifier).not.toHaveBeenCalled();
  });

  test("no identifier: nothing, and no fetch", () => {
    const { result } = renderHook(() => usePrincipal(undefined));
    expect(result.current).toBeUndefined();
    expect(store.getOrFetchUserByIdentifier).not.toHaveBeenCalled();
    expect(store.fetchServiceAccount).not.toHaveBeenCalled();
    expect(store.fetchWorkloadIdentity).not.toHaveBeenCalled();
  });
});
