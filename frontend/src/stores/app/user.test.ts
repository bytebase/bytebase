// @vitest-environment node
import { create } from "@bufbuild/protobuf";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { buildUserFilter, createUserSlice } from "./user";

const mocks = vi.hoisted(() => ({
  batchGetUsers: vi.fn(),
  getUser: vi.fn(),
}));
vi.mock("@/api", () => ({
  userServiceClientConnect: mocks,
}));

const QUOTED = 'SELECT * FROM "users"';
const ESCAPED = 'SELECT * FROM \\"users\\"';

describe("buildUserFilter", () => {
  test("escapes a quote in the free-text query", () => {
    const filter = buildUserFilter({ query: QUOTED }).toLowerCase();
    expect(filter).not.toContain(QUOTED.toLowerCase());
    expect(filter).toContain(ESCAPED.toLowerCase());
  });
});

describe("fetching users", () => {
  const person = create(UserSchema, {
    name: "users/alice@example.com",
    email: "alice@example.com",
    title: "Alice",
  });
  const serviceAccount = "users/deploy@service.bytebase.com";
  const workloadIdentity = "users/ci@workload.bytebase.com";

  const createStore = () => {
    const state: Record<string, unknown> = {};
    const set = (updater: unknown) => {
      Object.assign(
        state,
        typeof updater === "function" ? updater(state) : updater
      );
    };
    Object.assign(
      state,
      createUserSlice(set as never, (() => state) as never, {} as never)
    );
    return state as ReturnType<typeof createUserSlice>;
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("a batch that names a service account still resolves the people in it", async () => {
    mocks.batchGetUsers.mockResolvedValue({ users: [person] });
    const store = createStore();

    const users = await store.batchGetOrFetchUsers([
      person.name,
      serviceAccount,
      workloadIdentity,
    ]);

    expect(mocks.batchGetUsers).toHaveBeenCalledTimes(1);
    expect(mocks.batchGetUsers.mock.calls[0][0].names).toEqual([person.name]);
    expect(users.map((user) => user.title)).toEqual(["Alice", "deploy", "ci"]);
  });

  test("a batch of only service accounts sends nothing", async () => {
    const store = createStore();

    await store.batchGetOrFetchUsers([serviceAccount, workloadIdentity]);

    expect(mocks.batchGetUsers).not.toHaveBeenCalled();
    expect(mocks.getUser).not.toHaveBeenCalled();
  });

  test("a single service account is not asked of the user service", async () => {
    const store = createStore();

    expect(await store.fetchUser(serviceAccount)).toBeUndefined();
    expect(
      (await store.getOrFetchUserByIdentifier({ identifier: workloadIdentity }))
        .email
    ).toBe("ci@workload.bytebase.com");
    expect(mocks.getUser).not.toHaveBeenCalled();
  });
});
