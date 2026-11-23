import { create } from "@bufbuild/protobuf";
import { describe, expect, test } from "vitest";
import { AccessGrantSchema } from "@/types/proto-es/v1/access_grant_service_pb";
import {
  MaskingExemptionPolicy_ExemptionSchema,
  MaskingRulePolicy_MaskingRuleSchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  actionReceiptKey,
  createAccessGrantActionReceipt,
  createMaskingExemptionActionReceipt,
  createMaskingRuleActionReceipt,
  mergeActionReceipts,
  normalizeActionReceipts,
} from "./actionReceipt";

const policy = "workspaces/example/policies/MASKING_RULE";
const rule = create(MaskingRulePolicy_MaskingRuleSchema, {
  id: "rule-1",
  semanticType: "ssn",
  condition: { title: "Edited", expression: "true" },
});

describe("action receipts", () => {
  test("captures actual saved rule and exemption fields as protobuf JSON", () => {
    const receipt = createMaskingRuleActionReceipt(
      policy,
      rule,
      "draft-1",
      100
    )!;
    expect(receipt).toMatchObject({
      kind: "masking-rule",
      savedAt: 100,
      draftId: "draft-1",
      rule: { id: "rule-1", condition: { title: "Edited" } },
    });
    const exemption = createMaskingExemptionActionReceipt(
      "projects/example/policies/MASKING_EXEMPTION",
      create(MaskingExemptionPolicy_ExemptionSchema, {
        members: ["user:member@example.com"],
        condition: {
          expression: "request.time < timestamp('2030-01-01T00:00:00Z')",
        },
      }),
      "exemption-1",
      101
    );
    expect(exemption).toMatchObject({
      kind: "masking-exemption",
      savedAt: 101,
      exemption: { members: ["user:member@example.com"] },
    });
    rule.condition!.title = "Later editor mutation";
    expect(receipt.rule).toMatchObject({ condition: { title: "Edited" } });
  });

  test("migrates legacy Access Grant receipts and validates source provenance", () => {
    const grant = create(AccessGrantSchema, {
      name: "projects/example/accessGrants/1",
      creator: "users/member@example.com",
      query: "SELECT 2",
      targets: ["instances/example/databases/db"],
    });
    const receipt = createAccessGrantActionReceipt(grant, 100)!;
    const metadata = {
      accessRequestDraft: {
        project: "projects/example",
        sourceDatabase: "instances/example/databases/db",
        targets: grant.targets,
        statement: "SELECT 1",
      },
      submittedAccessGrants: [
        {
          grant: receipt.kind === "access-grant" ? receipt.grant : {},
          submittedAt: 100,
        },
      ],
    };
    expect(
      normalizeActionReceipts(undefined, metadata, "member@example.com")
    ).toEqual([receipt]);
    expect(
      normalizeActionReceipts([receipt], metadata, "other@example.com")
    ).toBeUndefined();
    expect(normalizeActionReceipts([receipt], undefined)).toBeUndefined();
  });

  test("requires the matching draft action and drops malformed observations", () => {
    const receipt = createMaskingRuleActionReceipt(
      policy,
      rule,
      "draft-1",
      100
    )!;
    const metadata = {
      actions: [{ kind: "draft-masking-rule" as const, draftId: "draft-1" }],
    };
    expect(normalizeActionReceipts([receipt], metadata)).toEqual([receipt]);
    expect(normalizeActionReceipts([receipt], { actions: [] })).toBeUndefined();
    for (const actions of ["invalid", {}, [null], ["draft-masking-rule"]]) {
      expect(
        normalizeActionReceipts([receipt], { actions } as never)
      ).toBeUndefined();
    }
    expect(
      normalizeActionReceipts(
        [{ ...receipt, observation: { status: "ACTIVE", observedAt: 200 } }],
        metadata
      )
    ).toEqual([receipt]);
    expect(
      normalizeActionReceipts(
        [{ ...receipt, observation: { status: "SAVED", observedAt: 99 } }],
        metadata
      )
    ).toEqual([receipt]);
  });

  test("merges latest observations without dropping other saved outcomes", () => {
    const receipt = createMaskingRuleActionReceipt(
      policy,
      rule,
      "draft-1",
      100
    )!;
    const second = createMaskingRuleActionReceipt(
      policy,
      { ...rule, id: "rule-2" },
      "draft-1",
      100
    )!;
    const refreshed = {
      ...receipt,
      observation: { status: "CHANGED" as const, observedAt: 300 },
    };
    expect(
      mergeActionReceipts([
        refreshed,
        receipt,
        second,
        { ...receipt, observation: { status: "SAVED", observedAt: 200 } },
      ])
    ).toEqual([refreshed, second]);
    expect(actionReceiptKey(second)).not.toBe(actionReceiptKey(receipt));
  });

  test.each([false, true])(
    "keeps distinct exemption saves from the same draft when configuration is identical: %s",
    (identical) => {
      const policy = "projects/example/policies/MASKING_EXEMPTION";
      const exemption = create(MaskingExemptionPolicy_ExemptionSchema, {
        members: ["user:member@example.com"],
        condition: { expression: "true" },
      });
      const first = createMaskingExemptionActionReceipt(
        policy,
        exemption,
        "draft-1",
        100
      )!;
      const second = createMaskingExemptionActionReceipt(
        policy,
        {
          ...exemption,
          members: identical ? exemption.members : ["user:other@example.com"],
        },
        "draft-1",
        100
      )!;
      expect(actionReceiptKey(second)).not.toBe(actionReceiptKey(first));
      expect(first.saveId).toEqual(expect.any(String));
      expect(second.saveId).not.toBe(first.saveId);
      const metadata = {
        actions: [
          { kind: "draft-masking-exemption" as const, draftId: "draft-1" },
        ],
      };
      const saved = JSON.parse(JSON.stringify([first, second, first]));
      expect(normalizeActionReceipts(saved, metadata)).toEqual([first, second]);
      expect(mergeActionReceipts([second, first, second])).toEqual([
        second,
        first,
      ]);
      const checked = {
        ...second,
        observation: { status: "MATCHING" as const, observedAt: 200 },
      };
      expect(mergeActionReceipts([first, checked, second])).toEqual([
        first,
        checked,
      ]);
    }
  );

  test("keeps legacy exemption keys stable and rejects invalid save identities", () => {
    const legacy = {
      kind: "masking-exemption" as const,
      policy: "projects/example/policies/MASKING_EXEMPTION",
      draftId: "draft-1",
      savedAt: 100,
      exemption: {
        members: ["user:member@example.com"],
        condition: { expression: "true" },
      },
    };
    const metadata = {
      actions: [
        { kind: "draft-masking-exemption" as const, draftId: "draft-1" },
      ],
    };
    const reloaded = normalizeActionReceipts(
      JSON.parse(JSON.stringify([legacy])),
      metadata
    )!;
    expect(actionReceiptKey(reloaded[0])).toBe(actionReceiptKey(legacy));
    expect(normalizeActionReceipts([legacy, ...reloaded], metadata)).toEqual([
      legacy,
    ]);
    for (const saveId of [null, 42, "", " "]) {
      expect(
        normalizeActionReceipts([{ ...legacy, saveId }], metadata)
      ).toBeUndefined();
    }
  });
});
