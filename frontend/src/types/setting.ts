export type SettingName =
  | "bb.branding.logo"
  | "bb.workspace.id"
  | "bb.app.im"
  | "bb.workspace.watermark"
  | "bb.workspace.profile"
  | "bb.workspace.approval"
  | "bb.workspace.approval.external"
  | "bb.plugin.openai.key"
  | "bb.plugin.openai.endpoint"
  | "bb.enterprise.trial"
  | "bb.workspace.schema-template"
  | "bb.workspace.data-classification"
  | "bb.workspace.semantic-types"
  | "bb.workspace.masking-algorithm"
  | "bb.workspace.maximum-sql-result-size"
  | "bb.workspace.scim";

export const defaultTokenDurationInHours = 7 * 24;
