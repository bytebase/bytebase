import { beforeEach, describe, expect, test, vi } from "vitest";
import type { SavedQuery } from "@/types/proto-es/v1/saved_query_service_pb";
import { SavedQueryBinding_Level } from "@/types/proto-es/v1/saved_query_service_pb";

const mocks = vi.hoisted(() => ({
  currentUser: { email: "me@example.com" },
  level: 0 as SavedQueryBinding_Level,
  hasProjectPermissionV2: vi.fn(
    (_project: unknown, _permission: string): boolean => false
  ),
}));

vi.mock("@/stores", () => ({
  getCurrentUserV1: () => mocks.currentUser,
}));

vi.mock("@/stores/app/projectAccess", () => ({
  getProjectByName: (name: string) => ({ name }),
}));

vi.mock("@/stores/app/savedQueryAccess", () => ({
  getSavedQueryLevel: () => mocks.level,
}));

vi.mock("@/utils", () => ({
  hasProjectPermissionV2: mocks.hasProjectPermissionV2,
}));

import {
  isSavedQueryDeletableV1,
  isSavedQueryReadableV1,
  isSavedQueryShareableV1,
  isSavedQueryWritableV1,
} from "./savedQuery";

const sheet = (creator: string) =>
  ({
    name: "projects/proj/savedQueries/sq",
    project: "projects/proj",
    creator,
  }) as SavedQuery;

const mine = sheet("users/me@example.com");
const theirs = sheet("users/other@example.com");

describe("saved query access predicates", () => {
  beforeEach(() => {
    mocks.level = SavedQueryBinding_Level.LEVEL_UNSPECIFIED;
    mocks.hasProjectPermissionV2.mockReset();
    mocks.hasProjectPermissionV2.mockReturnValue(false);
  });

  test("the creator holds every predicate", () => {
    expect(isSavedQueryReadableV1(mine)).toBe(true);
    expect(isSavedQueryWritableV1(mine)).toBe(true);
    expect(isSavedQueryDeletableV1(mine)).toBe(true);
    expect(isSavedQueryShareableV1(mine)).toBe(true);
  });

  test("a VIEWER binding grants read only", () => {
    mocks.level = SavedQueryBinding_Level.VIEWER;
    expect(isSavedQueryReadableV1(theirs)).toBe(true);
    expect(isSavedQueryWritableV1(theirs)).toBe(false);
    expect(isSavedQueryDeletableV1(theirs)).toBe(false);
    expect(isSavedQueryShareableV1(theirs)).toBe(false);
  });

  test("an EDITOR binding grants read and write, never delete or share", () => {
    mocks.level = SavedQueryBinding_Level.EDITOR;
    expect(isSavedQueryReadableV1(theirs)).toBe(true);
    expect(isSavedQueryWritableV1(theirs)).toBe(true);
    expect(isSavedQueryDeletableV1(theirs)).toBe(false);
    expect(isSavedQueryShareableV1(theirs)).toBe(false);
  });

  test("each predicate asks for exactly its project-level verb", () => {
    const byVerb = (verb: string) =>
      mocks.hasProjectPermissionV2.mockImplementation(
        (_project: unknown, permission: string) => permission === verb
      );

    byVerb("bb.savedQueries.get");
    expect(isSavedQueryReadableV1(theirs)).toBe(true);
    expect(isSavedQueryWritableV1(theirs)).toBe(false);

    byVerb("bb.savedQueries.update");
    expect(isSavedQueryWritableV1(theirs)).toBe(true);
    expect(isSavedQueryDeletableV1(theirs)).toBe(false);

    byVerb("bb.savedQueries.delete");
    expect(isSavedQueryDeletableV1(theirs)).toBe(true);
    expect(isSavedQueryShareableV1(theirs)).toBe(false);

    byVerb("bb.savedQueries.setIamPolicy");
    expect(isSavedQueryShareableV1(theirs)).toBe(true);
    expect(isSavedQueryReadableV1(theirs)).toBe(false);
  });
});
