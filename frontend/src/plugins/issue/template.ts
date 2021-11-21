import { IssueType } from "../../types";
import { FieldID, FieldInfo, IssueTemplate } from "../types";
import DatabaseCreateTemplate from "./DatabaseCreateTemplate";
import DatabaseGrantTemplate from "./DatabaseGrantTemplate";
import DatabaseSchemaBaselineTemplate from "./DatabaseSchemaBaselineTemplate";
import DatabaseSchemaUpdateTemplate from "./DatabaseSchemaUpdateTemplate";
import DefaultTemplate from "./DefaultTemplate";

type TemplateType = IssueType | "bb.issue.database.schema.baseline";

const allIssueTemplateList: IssueTemplate[] = [
  DefaultTemplate,
  DatabaseCreateTemplate,
  DatabaseGrantTemplate,
  DatabaseSchemaUpdateTemplate,
  DatabaseSchemaBaselineTemplate,
];

export function defaulTemplate(): IssueTemplate {
  return DefaultTemplate;
}

export function templateForType(type: TemplateType): IssueTemplate | undefined {
  return allIssueTemplateList.find((template) => template.type === type);
}

export function fieldInfoFromID(
  template: IssueTemplate,
  fieldID: FieldID
): FieldInfo {
  if (template.inputFieldList) {
    const field = template.inputFieldList.find((field) => field.id == fieldID);
    if (field) {
      return {
        name: field.name,
        type: field.type,
      };
    }
  }

  if (template.outputFieldList) {
    const field = template.outputFieldList.find((field) => field.id == fieldID);
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
