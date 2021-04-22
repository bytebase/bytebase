import { FieldId, IssueTemplate, FieldInfo } from "../types";
import { IssueType } from "../../types";
import DefaultTemplate from "./DefaultTemplate";
import DatabaseCreateTemplate from "./DatabaseCreateTemplate";
import DatabaseGrantTemplate from "./DatabaseGrantTemplate";
import DatabaseSchemaUpdateTemplate from "./DatabaseSchemaUpdateTemplate";

const allIssueTemplateList: IssueTemplate[] = [
  DefaultTemplate,
  DatabaseCreateTemplate,
  DatabaseGrantTemplate,
  DatabaseSchemaUpdateTemplate,
];

export function defaulTemplate(): IssueTemplate {
  return DefaultTemplate;
}

export function templateForType(type: IssueType): IssueTemplate | undefined {
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
