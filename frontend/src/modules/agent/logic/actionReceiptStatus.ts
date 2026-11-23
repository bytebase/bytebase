import { create, fromJson, type JsonObject, toJson } from "@bufbuild/protobuf";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { isEqual } from "lodash-es";
import {
  accessGrantServiceClientConnect,
  issueServiceClientConnect,
  orgPolicyServiceClientConnect,
} from "@/api";
import { silentContextKey } from "@/api/context-key";
import { buildAccessGrantFilter } from "@/stores/app/accessGrant";
import {
  AccessGrant_Status,
  AccessGrantSchema,
  SearchMyAccessGrantsRequestSchema,
} from "@/types/proto-es/v1/access_grant_service_pb";
import {
  GetIssueRequestSchema,
  type Issue,
  IssueSchema,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  GetPolicyRequestSchema,
  type MaskingExemptionPolicy_Exemption,
  MaskingExemptionPolicy_ExemptionSchema,
  MaskingRulePolicy_MaskingRuleSchema,
  type Policy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { getAccessGrantDisplayStatus } from "@/utils/accessGrant";
import { decodeAccessGrantActionReceipt } from "./actionReceipt";
import { decodeSubmittedAccessGrant } from "./submittedAccessGrant";
import type { AgentActionObservation, AgentActionReceipt } from "./types";

const normalizeExemption = (exemption: MaskingExemptionPolicy_Exemption) => ({
  ...(toJson(MaskingExemptionPolicy_ExemptionSchema, exemption) as JsonObject),
  members: [...exemption.members].sort((left, right) =>
    left.localeCompare(right)
  ),
});

export async function checkActionReceiptStatus(
  receipt: AgentActionReceipt,
  isCurrent: () => boolean = () => true
): Promise<AgentActionObservation> {
  const assertCurrent = () => {
    if (!isCurrent()) throw new Error("The saved action source has changed.");
  };
  const observedAt = () => Math.max(Date.now(), receipt.savedAt);
  assertCurrent();
  const options = {
    contextValues: createContextValues().set(silentContextKey, true),
  };
  if (receipt.kind === "access-grant") {
    const saved = decodeAccessGrantActionReceipt(receipt);
    if (!saved) throw new Error("The saved Access Grant is unavailable.");
    const project = saved.name.split("/").slice(0, 2).join("/");
    const response = await accessGrantServiceClientConnect.searchMyAccessGrants(
      create(SearchMyAccessGrantsRequestSchema, {
        parent: project,
        pageSize: 1,
        filter: buildAccessGrantFilter({ name: saved.name }),
      }),
      options
    );
    assertCurrent();
    const refreshed =
      response.accessGrants.length === 1
        ? decodeSubmittedAccessGrant({
            grant: toJson(AccessGrantSchema, response.accessGrants[0]!),
            submittedAt: observedAt(),
          })
        : undefined;
    if (
      !refreshed ||
      refreshed.name !== saved.name ||
      refreshed.creator !== saved.creator
    ) {
      throw new Error("The saved Access Grant is unavailable.");
    }
    let issue: Issue | undefined;
    if (refreshed.status === AccessGrant_Status.PENDING && refreshed.issue) {
      issue = await issueServiceClientConnect.getIssue(
        create(GetIssueRequestSchema, { name: refreshed.issue }),
        options
      );
      assertCurrent();
      if (issue.name !== refreshed.issue)
        throw new Error("The Access Grant issue is unavailable.");
    }
    return {
      status: getAccessGrantDisplayStatus(refreshed, issue),
      observedAt: observedAt(),
      grant: toJson(AccessGrantSchema, refreshed) as JsonObject,
      ...(issue ? { issue: toJson(IssueSchema, issue) as JsonObject } : {}),
    };
  }

  let policy: Policy;
  try {
    policy = await orgPolicyServiceClientConnect.getPolicy(
      create(GetPolicyRequestSchema, { name: receipt.policy }),
      options
    );
  } catch (error) {
    assertCurrent();
    if (ConnectError.from(error).code === Code.NotFound) {
      return { status: "NOT_FOUND", observedAt: observedAt() };
    }
    throw error;
  }
  assertCurrent();
  const parent = receipt.policy.slice(0, receipt.policy.lastIndexOf("/"));
  const policyType =
    receipt.kind === "masking-rule" ? "MASKING_RULE" : "MASKING_EXEMPTION";
  if (
    policy.name !== `${parent}/${policyType}` &&
    policy.name !== `${parent}/${policyType.toLowerCase()}`
  )
    throw new Error("The saved action policy is unavailable.");
  if (receipt.kind === "masking-rule") {
    if (policy.policy?.case !== "maskingRulePolicy")
      throw new Error("The masking rule policy is unavailable.");
    const saved = fromJson(MaskingRulePolicy_MaskingRuleSchema, receipt.rule);
    const current = policy.policy.value.rules.find(
      (rule) => rule.id === saved.id
    );
    return {
      status: !current
        ? "NOT_FOUND"
        : isEqual(
              toJson(MaskingRulePolicy_MaskingRuleSchema, current),
              toJson(MaskingRulePolicy_MaskingRuleSchema, saved)
            )
          ? "SAVED"
          : "CHANGED",
      observedAt: observedAt(),
    };
  }
  if (policy.policy?.case !== "maskingExemptionPolicy")
    throw new Error("The masking exemption policy is unavailable.");
  const saved = normalizeExemption(
    fromJson(MaskingExemptionPolicy_ExemptionSchema, receipt.exemption)
  );
  return {
    status: policy.policy.value.exemptions.some((exemption) =>
      isEqual(normalizeExemption(exemption), saved)
    )
      ? "MATCHING"
      : "NOT_FOUND",
    observedAt: observedAt(),
  };
}
