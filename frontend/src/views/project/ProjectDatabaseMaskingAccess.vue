<template>
  <MaskingExceptionUserTable
    size="medium"
    :access-list="accessList"
    :disabled="false"
    :show-database-column="true"
  />
</template>

<script lang="tsx" setup>
import { orderBy } from "lodash-es";
import { computed } from "vue";
import MaskingExceptionUserTable from "@/components/SensitiveData/MaskingExceptionUserTable.vue";
import type { AccessUser } from "@/components/SensitiveData/types";
import { useProjectByName } from "@/store";
import {
  usePolicyByParentAndType,
  useUserStore,
  useGroupStore,
  extractGroupEmail,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  getUserEmailInBinding,
  getGroupEmailInBinding,
  groupBindingPrefix,
} from "@/types";
import { maskingLevelToJSON } from "@/types/proto/v1/common";
import { PolicyType } from "@/types/proto/v1/org_policy_service";
import type { MaskingExceptionPolicy_MaskingException } from "@/types/proto/v1/org_policy_service";

const props = defineProps<{
  projectId: string;
}>();

const userStore = useUserStore();
const groupStore = useGroupStore();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);

const policy = usePolicyByParentAndType(
  computed(() => ({
    parentPath: project.value.name,
    policyType: PolicyType.MASKING_EXCEPTION,
  }))
);

const expirationTimeRegex = /request.time < timestamp\("(.+)?"\)/;

const getAccessUsers = (
  exception: MaskingExceptionPolicy_MaskingException
): AccessUser | undefined => {
  let expirationTimestamp: number | undefined;
  const expression = exception.condition?.expression ?? "";
  const matches = expirationTimeRegex.exec(expression);
  if (matches) {
    expirationTimestamp = new Date(matches[1]).getTime();
  }

  const access: AccessUser = {
    type: "user",
    key: exception.member,
    maskingLevel: exception.maskingLevel,
    expirationTimestamp,
    supportActions: new Set([exception.action]),
    rawExpression: expression,
  };

  if (exception.member.startsWith(groupBindingPrefix)) {
    access.type = "group";
    access.group = groupStore.getGroupByIdentifier(exception.member);
  } else {
    access.type = "user";
    access.user = userStore.getUserByIdentifier(exception.member);
  }

  if (!access.group && !access.user) {
    return;
  }

  return access;
};

const getExceptionIdentifier = (
  exception: MaskingExceptionPolicy_MaskingException
): string => {
  const expression = exception.condition?.expression ?? "";
  const res: string[] = [
    `level:"${maskingLevelToJSON(exception.maskingLevel)}"`,
    expression,
  ];
  return res.join(" && ");
};

const getMemberBinding = (access: AccessUser): string => {
  if (access.type === "user") {
    return getUserEmailInBinding(access.user!.email);
  }
  const email = extractGroupEmail(access.group!.name);
  return getGroupEmailInBinding(email);
};

const accessList = computed(() => {
  if (!policy.value || !policy.value.maskingExceptionPolicy) {
    return [];
  }

  const memberMap = new Map<string, AccessUser>();
  for (const exception of policy.value.maskingExceptionPolicy
    .maskingExceptions) {
    const identifier = getExceptionIdentifier(exception);
    const item = getAccessUsers(exception);
    if (!item) {
      continue;
    }
    const id = `${getMemberBinding(item)}:${identifier}`;
    item.key = id;
    const target = memberMap.get(id) ?? item;
    if (memberMap.has(id)) {
      for (const action of item.supportActions) {
        target.supportActions.add(action);
      }
    }
    memberMap.set(id, target);
  }

  return orderBy(
    [...memberMap.values()],
    [
      (access) => (access.type === "user" ? 1 : 0),
      (access) => {
        if (access.group) {
          return access.group.name;
        } else if (access.user) {
          return access.user.name;
        }
        return "";
      },
    ],
    ["desc", "desc"]
  );
});
</script>
