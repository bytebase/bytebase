import "@testing-library/jest-dom/vitest";
import { vi } from "vitest";

vi.mock("pouchdb", () => {
  class MockPouchDB {
    static plugin = vi.fn();

    get = vi.fn(async () => undefined);
    put = vi.fn(async () => undefined);
    remove = vi.fn(async () => undefined);
  }

  return {
    default: MockPouchDB,
  };
});

vi.mock("pouchdb-find", () => ({
  default: {},
}));
