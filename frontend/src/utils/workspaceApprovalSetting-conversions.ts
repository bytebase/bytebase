import type { ApprovalTemplate as OldApprovalTemplate, ApprovalNode as OldApprovalNode } from "@/types/proto/v1/issue_service";
import type { ApprovalTemplate as NewApprovalTemplate, ApprovalNode as NewApprovalNode } from "@/types/proto-es/v1/issue_service_pb";
import { ApprovalTemplateSchema as NewApprovalTemplateSchema, ApprovalNodeSchema as NewApprovalNodeSchema } from "@/types/proto-es/v1/issue_service_pb";
import { ApprovalTemplate as OldApprovalTemplateProto, ApprovalNode as OldApprovalNodeProto } from "@/types/proto/v1/issue_service";
import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Expr as OldExpr } from "@/types/proto/google/type/expr";
import type { Expr as NewExpr } from "@/types/proto-es/google/type/expr_pb";
import { ExprSchema as NewExprSchema } from "@/types/proto-es/google/type/expr_pb";
import { Expr as OldExprProto } from "@/types/proto/google/type/expr";

// Convert old ApprovalTemplate to new proto-es ApprovalTemplate
export const convertOldApprovalTemplateToNew = (
  oldTemplate: OldApprovalTemplate
): NewApprovalTemplate => {
  const json = OldApprovalTemplateProto.toJSON(oldTemplate) as any;
  return fromJson(NewApprovalTemplateSchema, json);
};

// Convert new proto-es ApprovalTemplate to old ApprovalTemplate
export const convertNewApprovalTemplateToOld = (
  newTemplate: NewApprovalTemplate
): OldApprovalTemplate => {
  const json = toJson(NewApprovalTemplateSchema, newTemplate);
  return OldApprovalTemplateProto.fromJSON(json);
};

// Convert old ApprovalNode to new proto-es ApprovalNode
export const convertOldApprovalNodeToNew = (
  oldNode: OldApprovalNode
): NewApprovalNode => {
  const json = OldApprovalNodeProto.toJSON(oldNode) as any;
  return fromJson(NewApprovalNodeSchema, json);
};

// Convert new proto-es ApprovalNode to old ApprovalNode
export const convertNewApprovalNodeToOld = (
  newNode: NewApprovalNode
): OldApprovalNode => {
  const json = toJson(NewApprovalNodeSchema, newNode);
  return OldApprovalNodeProto.fromJSON(json);
};

// Convert old Expr to new proto-es Expr
export const convertOldExprToNew = (oldExpr: OldExpr): NewExpr => {
  const json = OldExprProto.toJSON(oldExpr) as any;
  return fromJson(NewExprSchema, json);
};

// Convert new proto-es Expr to old Expr
export const convertNewExprToOld = (newExpr: NewExpr): OldExpr => {
  const json = toJson(NewExprSchema, newExpr);
  return OldExprProto.fromJSON(json);
};

// For WorkspaceApprovalSetting conversion
import type { WorkspaceApprovalSetting as OldWorkspaceApprovalSetting } from "@/types/proto/v1/setting_service";
import type { WorkspaceApprovalSetting as NewWorkspaceApprovalSetting } from "@/types/proto-es/v1/setting_service_pb";
import { WorkspaceApprovalSettingSchema as NewWorkspaceApprovalSettingSchema } from "@/types/proto-es/v1/setting_service_pb";
import { WorkspaceApprovalSetting as OldWorkspaceApprovalSettingProto } from "@/types/proto/v1/setting_service";

// Convert old WorkspaceApprovalSetting to new proto-es WorkspaceApprovalSetting
export const convertOldWorkspaceApprovalSettingToNew = (
  oldSetting: OldWorkspaceApprovalSetting
): NewWorkspaceApprovalSetting => {
  const json = OldWorkspaceApprovalSettingProto.toJSON(oldSetting) as any;
  return fromJson(NewWorkspaceApprovalSettingSchema, json);
};

// Convert new proto-es WorkspaceApprovalSetting to old WorkspaceApprovalSetting
export const convertNewWorkspaceApprovalSettingToOld = (
  newSetting: NewWorkspaceApprovalSetting
): OldWorkspaceApprovalSetting => {
  const json = toJson(NewWorkspaceApprovalSettingSchema, newSetting);
  return OldWorkspaceApprovalSettingProto.fromJSON(json);
};