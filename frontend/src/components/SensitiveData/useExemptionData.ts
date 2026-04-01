import { create } from "@bufbuild/protobuf";
import type { ComputedRef } from "vue";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  composePolicyBindings,
  pushNotification,
  useGroupStore,
  usePolicyByParentAndType,
  usePolicyV1Store,
  useSettingV1Store,
} from "@/store";
import { groupBindingPrefix } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type { MaskingExemptionPolicy_Exemption } from "@/types/proto-es/v1/org_policy_service_pb";
import {
  MaskingExemptionPolicySchema,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import {
  batchConvertFromCELString,
  type ConditionExpression,
} from "@/utils/issue/cel";
import {
  getConditionExpression,
  groupByMember,
  parseExpirationTimestamp,
} from "./exemptionDataUtils";
import type { AccessUser, ExemptionGrant, ExemptionMember } from "./types";

interface LocalState {
  loading: boolean;
  processing: boolean;
  rawAccessList: AccessUser[];
}

function getAccessUsers(
  exemption: MaskingExemptionPolicy_Exemption,
  condition: ConditionExpression
): AccessUser[] {
  const expression = exemption.condition?.expression ?? "";
  const description = exemption.condition?.description ?? "";
  const expirationTimestamp = parseExpirationTimestamp(expression);
  const conditionExpression = getConditionExpression(expression);

  return exemption.members.map((member) => ({
    type: member.startsWith(groupBindingPrefix)
      ? ("group" as const)
      : ("user" as const),
    member,
    key: `${member}:${expression}.${description}`,
    expirationTimestamp,
    rawExpression: expression,
    description,
    databaseResources:
      condition.databaseResources && condition.databaseResources.length > 0
        ? condition.databaseResources
        : undefined,
    conditionExpression,
  }));
}

function rebuildExemptions(accessList: AccessUser[]) {
  const expressionsMap = new Map<
    string,
    { description: string; members: string[] }
  >();

  for (const accessUser of accessList) {
    const expressions = accessUser.rawExpression.split(" && ").filter((e) => e);
    const index = expressions.findIndex((exp) =>
      exp.startsWith("request.time")
    );
    if (index >= 0) {
      if (!accessUser.expirationTimestamp) {
        expressions.splice(index, 1);
      } else {
        expressions[index] = `request.time < timestamp("${new Date(
          accessUser.expirationTimestamp
        ).toISOString()}")`;
      }
    } else if (accessUser.expirationTimestamp) {
      expressions.push(
        `request.time < timestamp("${new Date(
          accessUser.expirationTimestamp
        ).toISOString()}")`
      );
    }
    const finalExpression = expressions.join(" && ");
    if (!expressionsMap.has(finalExpression)) {
      expressionsMap.set(finalExpression, {
        description: accessUser.description,
        members: [],
      });
    }
    expressionsMap.get(finalExpression)!.members.push(accessUser.member);
  }

  const exemptions = [];
  for (const [expression, { description, members }] of expressionsMap) {
    exemptions.push({
      members,
      condition: create(ExprSchema, { description, expression }),
    });
  }
  return exemptions;
}

export function useExemptionData(projectName: ComputedRef<string>) {
  const { t } = useI18n();
  // Ensure group store is initialized for composePolicyBindings
  useGroupStore();
  const policyStore = usePolicyV1Store();
  // Ensure classification config is loaded so LevelBadge can resolve level names
  const settingStore = useSettingV1Store();
  settingStore.getOrFetchSettingByName(
    Setting_SettingName.DATA_CLASSIFICATION,
    true
  );

  const state = reactive<LocalState>({
    loading: true,
    processing: false,
    rawAccessList: [],
  });

  const { policy, ready } = usePolicyByParentAndType(
    computed(() => ({
      parentPath: projectName.value,
      policyType: PolicyType.MASKING_EXEMPTION,
    }))
  );

  const updateAccessUserList = async () => {
    if (!ready.value) {
      return;
    }
    if (
      !policy.value ||
      policy.value.policy?.case !== "maskingExemptionPolicy"
    ) {
      state.rawAccessList = [];
      state.loading = false;
      return;
    }

    try {
      const memberMap = new Map<string, AccessUser>();
      const { exemptions } = policy.value.policy.value;
      const expressionList = exemptions.map((e) =>
        e.condition?.expression ? e.condition.expression : "true"
      );
      const conditionList = await batchConvertFromCELString(expressionList);

      await composePolicyBindings(exemptions, true);
      for (let i = 0; i < exemptions.length; i++) {
        const exemption = exemptions[i];
        const condition = conditionList[i];
        for (const item of getAccessUsers(exemption, condition)) {
          const uniqueKey = `${item.key}:${i}`;
          memberMap.set(uniqueKey, item);
        }
      }

      state.rawAccessList = [...memberMap.values()];
    } finally {
      state.loading = false;
    }
  };

  watch([ready, () => policy.value], () => updateAccessUserList(), {
    immediate: true,
  });

  const members = computed<ExemptionMember[]>(() =>
    groupByMember(state.rawAccessList)
  );

  const upsertPolicy = async () => {
    const pol = await policyStore.getOrFetchPolicyByParentAndType({
      parentPath: projectName.value,
      policyType: PolicyType.MASKING_EXEMPTION,
    });
    if (!pol) return;

    pol.policy = {
      case: "maskingExemptionPolicy",
      value: create(MaskingExemptionPolicySchema, {
        exemptions: rebuildExemptions(state.rawAccessList),
      }),
    };
    await policyStore.upsertPolicy({
      parentPath: projectName.value,
      policy: pol,
    });
  };

  const submit = async () => {
    if (state.processing) return;
    state.processing = true;
    try {
      await upsertPolicy();
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.updated"),
      });
    } finally {
      state.processing = false;
    }
  };

  const revokeGrant = async (
    member: ExemptionMember,
    grant: ExemptionGrant
  ) => {
    // Remove only the first matching AccessUser to avoid over-removing when
    // duplicate exemptions exist (same member + expression + description).
    const idx = state.rawAccessList.findIndex(
      (a) =>
        a.member === member.member &&
        a.rawExpression === grant.rawExpression &&
        (a.description || "") === grant.description
    );
    if (idx < 0) return;
    const [removed] = state.rawAccessList.splice(idx, 1);
    try {
      await submit();
    } catch {
      // Restore on failure so UI stays in sync with backend
      state.rawAccessList.splice(idx, 0, removed);
    }
  };

  const loading = computed(() => !ready.value || state.loading);

  return {
    members,
    loading,
    revokeGrant,
  };
}
