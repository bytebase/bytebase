import { fromJson, toJson } from "@bufbuild/protobuf";
import type { IamPolicy as OldIamPolicy, Binding as OldBinding } from "@/types/proto/v1/iam_policy";
import { IamPolicy as OldIamPolicyProto, Binding as OldBindingProto } from "@/types/proto/v1/iam_policy";
import type { IamPolicy as NewIamPolicy, Binding as NewBinding } from "@/types/proto-es/v1/iam_policy_pb";
import { IamPolicySchema, BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";

// Convert old proto IamPolicy to proto-es IamPolicy
export const convertOldIamPolicyToNew = (oldPolicy: OldIamPolicy): NewIamPolicy => {
  const json = OldIamPolicyProto.toJSON(oldPolicy) as any;
  return fromJson(IamPolicySchema, json);
};

// Convert proto-es IamPolicy to old proto IamPolicy
export const convertNewIamPolicyToOld = (newPolicy: NewIamPolicy): OldIamPolicy => {
  const json = toJson(IamPolicySchema, newPolicy) as any;
  return OldIamPolicyProto.fromJSON(json);
};

// Convert old proto Binding to proto-es Binding
export const convertOldBindingToNew = (oldBinding: OldBinding): NewBinding => {
  const json = OldBindingProto.toJSON(oldBinding) as any;
  return fromJson(BindingSchema, json);
};

// Convert proto-es Binding to old proto Binding
export const convertNewBindingToOld = (newBinding: NewBinding): OldBinding => {
  const json = toJson(BindingSchema, newBinding) as any;
  return OldBindingProto.fromJSON(json);
};