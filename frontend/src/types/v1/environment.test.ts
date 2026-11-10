// @vitest-environment node
import { describe, expect, test } from "vitest";
import {
  type Environment,
  isProtectedEnvironment,
  NULL_ENVIRONMENT_NAME,
  resolveEnvironment,
  unknownEnvironment,
} from "./environment";

const env = (name: string, title: string, tags?: Record<string, string>) =>
  ({ ...unknownEnvironment(), name, title, tags: tags ?? {} }) as Environment;

const list = [env("environments/prod", "Production")];

describe("resolveEnvironment", () => {
  test("a name in the list resolves to that environment", () => {
    expect(resolveEnvironment("environments/prod", list).title).toBe(
      "Production"
    );
  });

  test("a name the list no longer has is titled by its id", () => {
    const gone = resolveEnvironment("environments/gone", list);
    expect(gone.name).toBe("environments/gone");
    expect(gone.title).toBe("gone");
  });

  test("no name is the null environment", () => {
    expect(resolveEnvironment("", list).name).toBe(NULL_ENVIRONMENT_NAME);
  });
});

describe("isProtectedEnvironment", () => {
  test("needs both the tag and the feature", () => {
    const prod = env("environments/prod", "Production", {
      protected: "protected",
    });
    expect(isProtectedEnvironment(prod, true)).toBe(true);
    expect(isProtectedEnvironment(prod, false)).toBe(false);
    expect(isProtectedEnvironment(env("environments/dev", "Dev"), true)).toBe(
      false
    );
  });
});
