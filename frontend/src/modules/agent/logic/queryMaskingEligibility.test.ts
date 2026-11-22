// @vitest-environment node
import { beforeEach, describe, expect, test, vi } from "vitest";
import { resolveQueryMaskingEligibility } from "./queryMaskingEligibility";

const mocks = vi.hoisted(() => ({ getMaskingSetup: vi.fn() }));
vi.mock("./tools/maskingSetup", () => ({
  getMaskingSetup: mocks.getMaskingSetup,
}));

const workspaceName = "workspaces/example";
const database = "instances/primary/databases/app";
const allowedSetup = () => ({
  features: { masking: "available" },
  steps: [
    {
      id: "global-masking",
      configure: { status: "allowed" },
      readAccess: { status: "allowed" },
    },
    { id: "semantic-types", readAccess: { status: "allowed" } },
  ],
});

beforeEach(() => {
  vi.resetAllMocks();
  mocks.getMaskingSetup.mockResolvedValue(allowedSetup());
});

describe("resolveQueryMaskingEligibility", () => {
  test.each([
    "",
    "workspaces/",
    "projects/example",
    "workspaces/-",
    "workspaces/example/other",
  ])(
    "does not refresh eligibility for an invalid workspace: %s",
    async (workspace) => {
      expect(await resolveQueryMaskingEligibility(workspace, database)).toEqual(
        {
          database,
          allowed: false,
        }
      );
      expect(mocks.getMaskingSetup).not.toHaveBeenCalled();
    }
  );

  test("allows discovery without masking-policy, semantic-type, or project facets", async () => {
    expect(
      await resolveQueryMaskingEligibility(workspaceName, database)
    ).toEqual({
      database,
      allowed: true,
    });
    expect(mocks.getMaskingSetup).toHaveBeenCalledExactlyOnceWith({
      workspaceName,
    });
  });

  test.each(["unknown", "unavailable", undefined])(
    "omits discovery when the masking feature is %s",
    async (masking) => {
      mocks.getMaskingSetup.mockResolvedValue({
        ...allowedSetup(),
        features: { masking },
      });
      expect(
        await resolveQueryMaskingEligibility(workspaceName, database)
      ).toEqual({
        database,
        allowed: false,
      });
    }
  );

  test.each([
    { step: "global-masking", access: "configure" },
    { step: "global-masking", access: "readAccess" },
    { step: "semantic-types", access: "readAccess" },
  ])("requires positive $step $access evidence", async ({ step, access }) => {
    for (const status of ["denied", "unknown", "needs-scope", undefined]) {
      const setup = allowedSetup();
      mocks.getMaskingSetup.mockResolvedValue({
        ...setup,
        steps: setup.steps.map((candidate) =>
          candidate.id === step
            ? { ...candidate, [access]: { status } }
            : candidate
        ),
      });
      expect(
        await resolveQueryMaskingEligibility(workspaceName, database)
      ).toEqual({
        database,
        allowed: false,
      });
    }
  });

  test.each(["global-masking", "semantic-types"])(
    "omits discovery when the %s step is missing",
    async (missingStep) => {
      const setup = allowedSetup();
      mocks.getMaskingSetup.mockResolvedValue({
        ...setup,
        steps: setup.steps.filter((step) => step.id !== missingStep),
      });
      expect(
        await resolveQueryMaskingEligibility(workspaceName, database)
      ).toEqual({
        database,
        allowed: false,
      });
    }
  );

  test.each([undefined, null, {}, { features: {}, steps: [] }])(
    "fails closed for incomplete setup evidence: %j",
    async (setup) => {
      mocks.getMaskingSetup.mockResolvedValue(setup);
      expect(
        await resolveQueryMaskingEligibility(workspaceName, database)
      ).toEqual({
        database,
        allowed: false,
      });
    }
  );

  test("fails closed when eligibility refresh rejects", async () => {
    mocks.getMaskingSetup.mockRejectedValue(new Error("refresh failed"));
    expect(
      await resolveQueryMaskingEligibility(workspaceName, database)
    ).toEqual({
      database,
      allowed: false,
    });
  });
});
