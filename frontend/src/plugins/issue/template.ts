import { IssueField, FieldId, IssueTemplate, UNKNOWN_FIELD } from "../types";
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

export function fieldFromId(
  template: IssueTemplate,
  fieldId: FieldId
): IssueField {
  if (template.fieldList) {
    return (
      template.fieldList.find((field) => field.id == fieldId) || UNKNOWN_FIELD
    );
  }
  return UNKNOWN_FIELD;
}
