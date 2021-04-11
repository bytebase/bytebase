import { TaskField, FieldId, TaskTemplate, UNKNOWN_FIELD } from "../types";
import { TaskType } from "../../types";
import DefaultTemplate from "./DefaultTemplate";
import DatabaseCreateTemplate from "./DatabaseCreateTemplate";
import DatabaseGrantTemplate from "./DatabaseGrantTemplate";
import DatabaseSchemaUpdateTemplate from "./DatabaseSchemaUpdateTemplate";

const allTaskTemplateList: TaskTemplate[] = [
  DefaultTemplate,
  DatabaseCreateTemplate,
  DatabaseGrantTemplate,
  DatabaseSchemaUpdateTemplate,
];

export function defaulTemplate(): TaskTemplate {
  return DefaultTemplate;
}

export function templateForType(type: TaskType): TaskTemplate | undefined {
  return allTaskTemplateList.find((template) => template.type === type);
}

export function fieldFromId(
  template: TaskTemplate,
  fieldId: FieldId
): TaskField {
  if (template.fieldList) {
    return (
      template.fieldList.find((field) => field.id == fieldId) || UNKNOWN_FIELD
    );
  }
  return UNKNOWN_FIELD;
}
