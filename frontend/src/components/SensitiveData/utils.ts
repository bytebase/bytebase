import { usePolicyV1Store } from "@/store";
import {
  Policy,
  MaskData,
  PolicyType,
  PolicyResourceType,
  MaskingExceptionPolicy_MaskingException,
  MaskingExceptionPolicy_MaskingException_Action,
} from "@/types/proto/v1/org_policy_service";
import { SensitiveColumn } from "./types";

export const getMaskDataIdentifier = (maskData: MaskData): string => {
  return `${maskData.schema}.${maskData.table}.${maskData.column}`;
};

export const isCurrentColumnException = (
  exception: MaskingExceptionPolicy_MaskingException,
  maskData: MaskData
): boolean => {
  const expression = exception.condition?.expression ?? "";
  const matches = [
    `resource.table_name == "${maskData.table}"`,
    `resource.column_name == "${maskData.column}"`,
  ];
  if (maskData.schema) {
    matches.push(`resource.schema_name == "${maskData.schema}"`);
  }

  for (const match of matches) {
    if (!expression.includes(match)) {
      return false;
    }
  }
  return true;
};

export const removeSensitiveColumn = async (
  sensitiveColumn: SensitiveColumn
) => {
  const policyStore = usePolicyV1Store();
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: sensitiveColumn.database.name,
    policyType: PolicyType.MASKING,
  });
  if (!policy) return;

  const maskData = policy.maskingPolicy?.maskData;
  if (!maskData) return;

  const index = maskData.findIndex(
    (sensitiveData) =>
      getMaskDataIdentifier(sensitiveData) ===
      getMaskDataIdentifier(sensitiveColumn.maskData)
  );
  if (index >= 0) {
    // mutate the list and the item directly
    // so we don't need to re-fetch the whole list.
    maskData.splice(index, 1);

    await policyStore.updatePolicy(["payload"], {
      name: policy.name,
      type: PolicyType.MASKING,
      resourceType: PolicyResourceType.DATABASE,
      maskingPolicy: {
        maskData,
      },
    });
    await removeMaskingExceptions(sensitiveColumn);
  }
};

const removeMaskingExceptions = async (sensitiveColumn: SensitiveColumn) => {
  const policyStore = usePolicyV1Store();
  const policy = await policyStore.getOrFetchPolicyByParentAndType({
    parentPath: sensitiveColumn.database.name,
    policyType: PolicyType.MASKING_EXCEPTION,
  });
  if (!policy) {
    return;
  }

  const exceptions = (
    policy.maskingExceptionPolicy?.maskingExceptions ?? []
  ).filter(
    (exception) =>
      !isCurrentColumnException(exception, sensitiveColumn.maskData)
  );

  policy.maskingExceptionPolicy = {
    ...(policy.maskingExceptionPolicy ?? {}),
    maskingExceptions: exceptions,
  };
  await policyStore.updatePolicy(["payload"], policy);
};
