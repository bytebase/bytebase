import { IssueType } from "../../types";
import { FieldId, FieldInfo, IssueTemplate } from "../types";
import DatabaseCreateTemplate from "./DatabaseCreateTemplate";
import DatabaseGrantTemplate from "./DatabaseGrantTemplate";
import DatabaseSchemaBaselineTemplate from "./DatabaseSchemaBaselineTemplate";
import DatabaseSchemaUpdateTemplate from "./DatabaseSchemaUpdateTemplate";
import DatabaseDataUpdateTemplate from "./DatabaseDataUpdateTemplate";
import DefaultTemplate from "./DefaultTemplate";

export type TemplateType = IssueType | "bb.issue.database.schema.baseline";

const allIssueTemplateList: IssueTemplate[] = [
  DefaultTemplate,
  DatabaseCreateTemplate,
  DatabaseGrantTemplate,
  DatabaseSchemaUpdateTemplate,
  DatabaseDataUpdateTemplate,
  DatabaseSchemaBaselineTemplate,
];

export function defaultTemplate(): IssueTemplate {
  return DefaultTemplate;
}

export function templateForType(type: TemplateType): IssueTemplate | undefined {
  return allIssueTemplateList.find((template) => template.type === type);
}

export function fieldInfoFromId(
  template: IssueTemplate,
  fieldId: FieldId
): FieldInfo {
  if (template.inputFieldList) {
    const field = template.inputFieldList.find((field) => field.id == fieldId);
    if (field) {
      return {
        name: field.name,
        type: field.type,
      };
    }
  }

  if (template.outputFieldList) {
    const field = template.outputFieldList.find((field) => field.id == fieldId);
    if (field) {
      return {
        name: field.name,
        type: field.type,
      };
    }
  }
  return {
    name: "<<Unknown field>>",
    type: "String",
  };
}
