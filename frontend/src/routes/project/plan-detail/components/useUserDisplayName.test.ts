import { renderHook } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { useUserDisplayName } from "./useUserDisplayName";

const store = vi.hoisted(() => {
  const state = {
    usersByName: {} as Record<string, { title: string }>,
    serviceAccountsByName: {} as Record<string, { title: string }>,
    workloadIdentitiesByName: {} as Record<string, { title: string }>,
    getUserByIdentifier: (identifier: string) => state.usersByName[identifier],
    // As the store does: an unread account is named from its email.
    getServiceAccount: (name: string) =>
      state.serviceAccountsByName[name] ?? {
        title: name.slice("serviceAccounts/".length).split("@")[0],
      },
    getWorkloadIdentity: (name: string) =>
      state.workloadIdentitiesByName[name] ?? {
        title: name.slice("workloadIdentities/".length).split("@")[0],
      },
    getOrFetchUserByIdentifier: vi.fn(async () => ({})),
    fetchServiceAccount: vi.fn(async () => ({})),
    fetchWorkloadIdentity: vi.fn(async () => ({})),
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
  vi.clearAllMocks();
});

test("a person is named from the user service, fetched silently", () => {
  store.usersByName["users/alex@example.com"] = { title: "Alex Kim" };
  const { result } = renderHook(() =>
    useUserDisplayName("users/alex@example.com")
  );
  expect(result.current).toBe("Alex Kim");

  renderHook(() => useUserDisplayName("users/new@example.com"));
  expect(store.getOrFetchUserByIdentifier).toHaveBeenCalledWith({
    identifier: "users/new@example.com",
    silent: true,
  });
});

test.each([
  [
    "a service account",
    "ci@service.bytebase.com",
    "serviceAccountsByName",
    "serviceAccounts/ci@service.bytebase.com",
  ],
  [
    "a workload identity",
    "ci@workload.bytebase.com",
    "workloadIdentitiesByName",
    "workloadIdentities/ci@workload.bytebase.com",
  ],
] as const)(
  "%s, as CI is, is named from its own service",
  (_, email, cache, name) => {
    store[cache][name] = { title: "Shop CI" };
    const { result } = renderHook(() => useUserDisplayName(`users/${email}`));
    expect(result.current).toBe("Shop CI");
    expect(store.getOrFetchUserByIdentifier).not.toHaveBeenCalled();
  }
);

test("until its record loads, a person reads as the email, and CI as the name in its email, as the member list does", () => {
  expect(
    renderHook(() => useUserDisplayName("users/new@example.com")).result.current
  ).toBe("new@example.com");
  expect(
    renderHook(() => useUserDisplayName("users/ci@service.bytebase.com")).result
      .current
  ).toBe("ci");
});
