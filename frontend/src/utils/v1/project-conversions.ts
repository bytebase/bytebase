import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Project as OldProject, Webhook as OldWebhook } from "@/types/proto/v1/project_service";
import { Project as OldProjectProto, Webhook as OldWebhookProto } from "@/types/proto/v1/project_service";
import type { IamPolicy as OldIamPolicy } from "@/types/proto/v1/iam_policy";
import { IamPolicy as OldIamPolicyProto } from "@/types/proto/v1/iam_policy";
import type { Project as NewProject, Webhook as NewWebhook } from "@/types/proto-es/v1/project_service_pb";
import { ProjectSchema, WebhookSchema } from "@/types/proto-es/v1/project_service_pb";
import type { IamPolicy as NewIamPolicy } from "@/types/proto-es/v1/iam_policy_pb";
import { IamPolicySchema } from "@/types/proto-es/v1/iam_policy_pb";

// Convert old proto to proto-es
export const convertOldProjectToNew = (oldProject: OldProject): NewProject => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldProjectProto.toJSON(oldProject) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(ProjectSchema, json);
};

// Convert proto-es to old proto
export const convertNewProjectToOld = (newProject: NewProject): OldProject => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(ProjectSchema, newProject);
  return OldProjectProto.fromJSON(json);
};

// Convert old webhook proto to proto-es
export const convertOldWebhookToNew = (oldWebhook: OldWebhook): NewWebhook => {
  const json = OldWebhookProto.toJSON(oldWebhook) as any;
  return fromJson(WebhookSchema, json);
};

// Convert proto-es webhook to old proto
export const convertNewWebhookToOld = (newWebhook: NewWebhook): OldWebhook => {
  const json = toJson(WebhookSchema, newWebhook);
  return OldWebhookProto.fromJSON(json);
};

// Convert old IamPolicy proto to proto-es
export const convertOldIamPolicyToNew = (oldPolicy: OldIamPolicy): NewIamPolicy => {
  const json = OldIamPolicyProto.toJSON(oldPolicy) as any;
  return fromJson(IamPolicySchema, json);
};

// Convert proto-es IamPolicy to old proto
export const convertNewIamPolicyToOld = (newPolicy: NewIamPolicy): OldIamPolicy => {
  const json = toJson(IamPolicySchema, newPolicy);
  return OldIamPolicyProto.fromJSON(json);
};