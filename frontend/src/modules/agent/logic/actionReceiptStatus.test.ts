import { create, toJson } from "@bufbuild/protobuf";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";
import { Code, ConnectError } from "@connectrpc/connect";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { silentContextKey } from "@/api/context-key";
import {
  AccessGrant_Status,
  AccessGrantSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import { ApprovalStatus, IssueStatus } from "@/types/proto-es/v1/common_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import {
  MaskingExemptionPolicy_ExemptionSchema,
  MaskingRulePolicy_MaskingRuleSchema,
  PolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  createAccessGrantActionReceipt,
  createMaskingExemptionActionReceipt,
  createMaskingRuleActionReceipt,
} from "./actionReceipt";
import { checkActionReceiptStatus } from "./actionReceiptStatus";

const mocks = vi.hoisted(() => ({
  searchMyAccessGrants: vi.fn(),
  getIssue: vi.fn(),
  getPolicy: vi.fn(),
}));

vi.mock("@/api", () => ({
  accessGrantServiceClientConnect: {
    searchMyAccessGrants: mocks.searchMyAccessGrants,
  },
  issueServiceClientConnect: { getIssue: mocks.getIssue },
  orgPolicyServiceClientConnect: { getPolicy: mocks.getPolicy },
}));

const grant = create(AccessGrantSchema, {
  name: "projects/example/accessGrants/42",
  creator: "users/member@example.com",
  query: "SELECT 1",
  targets: ["instances/example/databases/app"],
  issue: "projects/example/issues/7",
  status: AccessGrant_Status.PENDING,
});
const grantReceipt = () => createAccessGrantActionReceipt(grant, 100)!;
const rule = create(MaskingRulePolicy_MaskingRuleSchema, {
  id: "rule-1",
  semanticType: "email",
  condition: {
    title: "Edited rule",
    expression: 'resource.column_name == "email"',
  },
});
const rulePolicy = "workspaces/example/policies/MASKING_RULE";
const ruleReceipt = () =>
  createMaskingRuleActionReceipt(rulePolicy, rule, "draft-1", 100)!;
const exemption = create(MaskingExemptionPolicy_ExemptionSchema, {
  members: ["user:member@example.com", "group:reviewers@example.com"],
  condition: {
    expression:
      'request.time < timestamp("2030-01-01T00:00:00Z") && resource.column_name == "email"',
    description: "Edited review access",
  },
});
const exemptionPolicy = "projects/example/policies/MASKING_EXEMPTION";
const exemptionReceipt = () =>
  createMaskingExemptionActionReceipt(
    exemptionPolicy,
    exemption,
    "draft-2",
    100
  )!;

beforeEach(() => {
  vi.clearAllMocks();
  vi.spyOn(Date, "now").mockReturnValue(1000);
  mocks.searchMyAccessGrants.mockResolvedValue({ accessGrants: [grant] });
  mocks.getIssue.mockResolvedValue(
    create(IssueSchema, { name: grant.issue, status: IssueStatus.OPEN })
  );
  mocks.getPolicy.mockResolvedValue(
    create(PolicySchema, {
      name: rulePolicy,
      policy: { case: "maskingRulePolicy", value: { rules: [rule] } },
    })
  );
});

afterEach(() => vi.restoreAllMocks());

describe("action receipt status", () => {
  test("does not read until explicitly invoked and performs a fresh exact grant lookup each time", async () => {
    expect(mocks.searchMyAccessGrants).not.toHaveBeenCalled();
    expect(mocks.getPolicy).not.toHaveBeenCalled();
    const receipt = grantReceipt();
    const observation = await checkActionReceiptStatus(receipt);
    expect(mocks.searchMyAccessGrants).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({
        parent: "projects/example",
        pageSize: 1,
        filter: 'name == "projects/example/accessGrants/42"',
      }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
    expect(
      mocks.searchMyAccessGrants.mock.calls[0][1].contextValues.get(
        silentContextKey
      )
    ).toBe(true);
    expect(mocks.getIssue).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ name: grant.issue }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
    expect(observation).toMatchObject({
      status: "PENDING",
      observedAt: 1000,
      grant: toJson(AccessGrantSchema, grant),
    });
    expect(observation.issue).toMatchObject({ name: grant.issue });
    await checkActionReceiptStatus(receipt);
    expect(mocks.searchMyAccessGrants).toHaveBeenCalledTimes(2);
  });

  test.each([
    [ApprovalStatus.REJECTED, IssueStatus.OPEN, "REJECTED"],
    [ApprovalStatus.PENDING, IssueStatus.CANCELED, "CANCELED"],
  ] as const)(
    "uses native issue approval status %s and lifecycle %s",
    async (approvalStatus, status, expected) => {
      mocks.getIssue.mockResolvedValue(
        create(IssueSchema, { name: grant.issue, approvalStatus, status })
      );
      expect(await checkActionReceiptStatus(grantReceipt())).toMatchObject({
        status: expected,
      });
    }
  );

  test.each([
    [AccessGrant_Status.ACTIVE, 10n, "ACTIVE"],
    [AccessGrant_Status.ACTIVE, 0n, "EXPIRED"],
    [AccessGrant_Status.REVOKED, 10n, "REVOKED"],
  ] as const)(
    "reads grant state %s with expiration %s without unnecessary issue lookup",
    async (status, seconds, expected) => {
      const current = create(AccessGrantSchema, {
        ...grant,
        status,
        expiration: {
          case: "expireTime",
          value: create(TimestampSchema, { seconds }),
        },
      });
      mocks.searchMyAccessGrants.mockResolvedValue({ accessGrants: [current] });
      expect(await checkActionReceiptStatus(grantReceipt())).toMatchObject({
        status: expected,
        grant: toJson(AccessGrantSchema, current),
      });
      expect(mocks.getIssue).not.toHaveBeenCalled();
    }
  );

  test.each([
    [],
    [grant, grant],
    [
      create(AccessGrantSchema, {
        ...grant,
        name: "projects/other/accessGrants/42",
      }),
    ],
    [
      create(AccessGrantSchema, {
        ...grant,
        creator: "users/other@example.com",
      }),
    ],
    [create(AccessGrantSchema, { ...grant, query: "" })],
  ])(
    "rejects an unavailable, foreign, or malformed grant: %j",
    async (...accessGrants) => {
      mocks.searchMyAccessGrants.mockResolvedValue({ accessGrants });
      await expect(checkActionReceiptStatus(grantReceipt())).rejects.toThrow();
      expect(mocks.getIssue).not.toHaveBeenCalled();
    }
  );

  test("rejects an issue belonging to a different request", async () => {
    mocks.getIssue.mockResolvedValue(
      create(IssueSchema, { name: "projects/example/issues/8" })
    );
    await expect(checkActionReceiptStatus(grantReceipt())).rejects.toThrow();
  });

  test("does not retry a denied grant read or mutate the saved snapshot", async () => {
    mocks.searchMyAccessGrants.mockRejectedValueOnce(
      new ConnectError("denied", Code.PermissionDenied)
    );
    const receipt = grantReceipt();
    const saved = JSON.stringify(receipt);
    await expect(checkActionReceiptStatus(receipt)).rejects.toThrow("denied");
    expect(mocks.searchMyAccessGrants).toHaveBeenCalledOnce();
    expect(JSON.stringify(receipt)).toBe(saved);
  });

  test("does not dispatch a read after source ownership is lost", async () => {
    await expect(
      checkActionReceiptStatus(grantReceipt(), () => false)
    ).rejects.toThrow();
    expect(mocks.searchMyAccessGrants).not.toHaveBeenCalled();
    expect(mocks.getIssue).not.toHaveBeenCalled();
  });

  test("does not read the issue after ownership changes during the grant check", async () => {
    let current = true;
    mocks.searchMyAccessGrants.mockImplementationOnce(async () => {
      current = false;
      return { accessGrants: [grant] };
    });
    await expect(
      checkActionReceiptStatus(grantReceipt(), () => current)
    ).rejects.toThrow();
    expect(mocks.searchMyAccessGrants).toHaveBeenCalledOnce();
    expect(mocks.getIssue).not.toHaveBeenCalled();
  });

  test("reads the exact global policy and finds the saved rule by stable ID", async () => {
    expect(await checkActionReceiptStatus(ruleReceipt())).toEqual({
      status: "SAVED",
      observedAt: 1000,
    });
    expect(mocks.getPolicy).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ name: rulePolicy }),
      expect.objectContaining({ contextValues: expect.anything() })
    );
    expect(
      mocks.getPolicy.mock.calls[0][1].contextValues.get(silentContextKey)
    ).toBe(true);
  });

  test.each([
    [
      [
        create(MaskingRulePolicy_MaskingRuleSchema, {
          ...rule,
          condition: { ...rule.condition!, title: "Renamed" },
        }),
      ],
      "CHANGED",
    ],
    [
      [
        create(MaskingRulePolicy_MaskingRuleSchema, {
          ...rule,
          semanticType: "phone",
        }),
      ],
      "CHANGED",
    ],
    [
      [
        create(MaskingRulePolicy_MaskingRuleSchema, {
          ...rule,
          id: "other-rule",
        }),
      ],
      "NOT_FOUND",
    ],
    [[], "NOT_FOUND"],
  ] as const)(
    "distinguishes changed and missing rules: %j",
    async (rules, expected) => {
      mocks.getPolicy.mockResolvedValue(
        create(PolicySchema, {
          name: rulePolicy,
          policy: { case: "maskingRulePolicy", value: { rules: [...rules] } },
        })
      );
      expect(await checkActionReceiptStatus(ruleReceipt())).toEqual({
        status: expected,
        observedAt: 1000,
      });
    }
  );

  test("matches the full saved exemption with member order normalized", async () => {
    mocks.getPolicy.mockResolvedValue(
      create(PolicySchema, {
        name: exemptionPolicy,
        policy: {
          case: "maskingExemptionPolicy",
          value: {
            exemptions: [
              { ...exemption, members: [...exemption.members].reverse() },
            ],
          },
        },
      })
    );
    expect(await checkActionReceiptStatus(exemptionReceipt())).toEqual({
      status: "MATCHING",
      observedAt: 1000,
    });
    expect(mocks.getPolicy.mock.calls[0][0].name).toBe(exemptionPolicy);
  });

  test.each([
    [
      ruleReceipt,
      create(PolicySchema, {
        name: "workspaces/example/policies/masking_rule",
        policy: { case: "maskingRulePolicy", value: { rules: [rule] } },
      }),
      "SAVED",
    ],
    [
      exemptionReceipt,
      create(PolicySchema, {
        name: "projects/example/policies/masking_exemption",
        policy: {
          case: "maskingExemptionPolicy",
          value: { exemptions: [exemption] },
        },
      }),
      "MATCHING",
    ],
  ] as const)(
    "accepts the backend's canonical lowercase policy resource",
    async (receipt, policy, expected) => {
      mocks.getPolicy.mockResolvedValueOnce(policy);
      expect(await checkActionReceiptStatus(receipt())).toEqual({
        status: expected,
        observedAt: 1000,
      });
    }
  );

  test.each([
    [
      ruleReceipt,
      create(PolicySchema, {
        name: "workspaces/other/policies/masking_rule",
        policy: { case: "maskingRulePolicy", value: { rules: [rule] } },
      }),
    ],
    [
      ruleReceipt,
      create(PolicySchema, {
        name: "workspaces/EXAMPLE/policies/masking_rule",
        policy: { case: "maskingRulePolicy", value: { rules: [rule] } },
      }),
    ],
    [
      ruleReceipt,
      create(PolicySchema, {
        name: "workspaces/example/policies/masking_exemption",
        policy: { case: "maskingRulePolicy", value: { rules: [rule] } },
      }),
    ],
    [
      exemptionReceipt,
      create(PolicySchema, {
        name: "projects/other/policies/masking_exemption",
        policy: {
          case: "maskingExemptionPolicy",
          value: { exemptions: [exemption] },
        },
      }),
    ],
  ] as const)(
    "does not accept a canonical policy from another parent or type",
    async (receipt, policy) => {
      mocks.getPolicy.mockResolvedValueOnce(policy);
      await expect(checkActionReceiptStatus(receipt())).rejects.toThrow();
    }
  );

  test.each([
    { members: ["user:other@example.com", "group:reviewers@example.com"] },
    {
      condition: {
        ...exemption.condition!,
        expression: exemption.condition!.expression.replace("2030", "2031"),
      },
    },
    {
      condition: {
        ...exemption.condition!,
        description: "Different exemption",
      },
    },
  ])(
    "does not substitute a partially matching exemption: %j",
    async (patch) => {
      mocks.getPolicy.mockResolvedValue(
        create(PolicySchema, {
          name: exemptionPolicy,
          policy: {
            case: "maskingExemptionPolicy",
            value: { exemptions: [{ ...exemption, ...patch }] },
          },
        })
      );
      expect(await checkActionReceiptStatus(exemptionReceipt())).toEqual({
        status: "NOT_FOUND",
        observedAt: 1000,
      });
    }
  );

  test.each([ruleReceipt, exemptionReceipt])(
    "treats a missing policy as not found without retry",
    async (receipt) => {
      mocks.getPolicy.mockRejectedValueOnce(
        new ConnectError("missing", Code.NotFound)
      );
      expect(await checkActionReceiptStatus(receipt())).toEqual({
        status: "NOT_FOUND",
        observedAt: 1000,
      });
      expect(mocks.getPolicy).toHaveBeenCalledOnce();
    }
  );

  test("does not publish a missing-policy observation after ownership changes", async () => {
    let current = true;
    mocks.getPolicy.mockImplementationOnce(async () => {
      current = false;
      throw new ConnectError("missing", Code.NotFound);
    });
    await expect(
      checkActionReceiptStatus(ruleReceipt(), () => current)
    ).rejects.toThrow();
    expect(mocks.getPolicy).toHaveBeenCalledOnce();
  });

  test("never timestamps an observation earlier than its saved receipt", async () => {
    vi.mocked(Date.now).mockReturnValue(50);
    expect(await checkActionReceiptStatus(ruleReceipt())).toEqual({
      status: "SAVED",
      observedAt: 100,
    });
  });

  test.each([Code.PermissionDenied, Code.Unavailable])(
    "keeps unreadable policy %s distinct from not found",
    async (code) => {
      mocks.getPolicy.mockRejectedValueOnce(
        new ConnectError("unreadable", code)
      );
      await expect(checkActionReceiptStatus(ruleReceipt())).rejects.toThrow(
        "unreadable"
      );
      expect(mocks.getPolicy).toHaveBeenCalledOnce();
    }
  );

  test.each([
    create(PolicySchema, {
      name: "workspaces/other/policies/MASKING_RULE",
      policy: { case: "maskingRulePolicy", value: { rules: [rule] } },
    }),
    create(PolicySchema, {
      name: rulePolicy,
      policy: { case: "maskingExemptionPolicy", value: { exemptions: [] } },
    }),
  ])(
    "rejects a policy with the wrong resource name or kind",
    async (policy) => {
      mocks.getPolicy.mockResolvedValueOnce(policy);
      await expect(checkActionReceiptStatus(ruleReceipt())).rejects.toThrow();
    }
  );
});
