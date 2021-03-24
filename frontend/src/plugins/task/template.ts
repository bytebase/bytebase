import {
  TaskField,
  TaskFieldId,
  TaskTemplate,
  TemplateContext,
  TaskBuiltinFieldId,
  DatabaseFieldPayload,
  UNKNOWN_FIELD,
} from "../types";
import { EnvironmentId, TaskType, TaskNew } from "../../types";
import DatabaseRequestTemplate from "./DatabaseRequestTemplate";
import DatabaseSchemaUpdateTemplate from "./DatabaseSchemaUpdateTemplate";

const DEFAULT_TEMPLATE = {
  type: "bytebase.general",
  buildTask: (ctx: TemplateContext): TaskNew => {
    return {
      name: "New General Task",
      type: "bytebase.general",
      description: "",
      stageProgressList: [
        {
          id: "1",
          name: "Request",
          type: "SIMPLE",
          status: "PENDING",
        },
      ],
      creatorId: ctx.currentUser.id,
      subscriberIdList: [],
      payload: {},
    };
  },
  fieldList: [],
};

const allTaskTemplateList: TaskTemplate[] = [
  DEFAULT_TEMPLATE,
  DatabaseRequestTemplate,
  DatabaseSchemaUpdateTemplate,
];

export function defaulTemplate(): TaskTemplate {
  return DEFAULT_TEMPLATE;
}

export function templateForType(type: TaskType): TaskTemplate | undefined {
  return allTaskTemplateList.find((template) => template.type === type);
}

export function fieldFromId(
  template: TaskTemplate,
  fieldId: TaskFieldId
): TaskField {
  if (template.fieldList) {
    return (
      template.fieldList.find((field) => field.id == fieldId) || UNKNOWN_FIELD
    );
  }
  return UNKNOWN_FIELD;
}
