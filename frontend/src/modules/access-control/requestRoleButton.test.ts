import { describe, expect, test } from "vitest";
import { getRequestRoleButtonState } from "./requestRoleButton";

describe("getRequestRoleButtonState", () => {
  const t = (key: string) => `translated:${key}`;

  const base = {
    t,
    projectName: "projects/demo",
    projectReady: true,
    allowRequestRole: true,
    hasFullProjectAccess: false,
    hasRequestRoleFeature: true,
  } as const;

  test("hides the button outside project scope", () => {
    expect(
      getRequestRoleButtonState({
        ...base,
        projectName: undefined,
      })
    ).toEqual({
      visible: false,
    });
  });

  test("shows a loading-disabled button while project state is unavailable", () => {
    expect(
      getRequestRoleButtonState({
        ...base,
        projectReady: false,
      })
    ).toEqual({
      visible: true,
      disabledReason: "translated:common.loading",
    });
  });

  test("shows a disabled button when request role is turned off for the project", () => {
    expect(
      getRequestRoleButtonState({
        ...base,
        allowRequestRole: false,
      })
    ).toEqual({
      visible: true,
      disabledReason:
        "translated:project.members.request-role.disabled-reason.allow-request-role-disabled",
    });
  });

  test("shows a disabled button when the user already has full project access", () => {
    expect(
      getRequestRoleButtonState({
        ...base,
        hasFullProjectAccess: true,
      })
    ).toEqual({
      visible: false,
    });
  });

  test("leaves permission-related disabling to the permission guard", () => {
    expect(
      getRequestRoleButtonState({
        ...base,
      })
    ).toEqual({
      visible: true,
    });
  });

  test("shows a disabled button when the workflow feature is unavailable", () => {
    expect(
      getRequestRoleButtonState({
        ...base,
        hasRequestRoleFeature: false,
      })
    ).toEqual({
      visible: true,
      disabledReason:
        "translated:project.members.request-role.disabled-reason.feature-unavailable",
    });
  });

  test("enables the button when every prerequisite is satisfied", () => {
    expect(getRequestRoleButtonState(base)).toEqual({
      visible: true,
    });
  });
});
