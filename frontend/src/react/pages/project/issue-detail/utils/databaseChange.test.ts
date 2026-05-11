import { describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { BACKUP_AVAILABLE_ENGINES } from "./databaseChange";

vi.mock("@/types", () => ({
  isValidDatabaseName: vi.fn(),
}));

vi.mock("@/utils", () => ({
  getInstanceResource: vi.fn(),
  semverCompare: vi.fn(),
}));

describe("BACKUP_AVAILABLE_ENGINES", () => {
  test("includes MariaDB", () => {
    expect(BACKUP_AVAILABLE_ENGINES).toContain(Engine.MARIADB);
  });
});
