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
import DatabaseCreateTemplate from "./DatabaseCreateTemplate";
import DatabaseSchemaUpdateTemplate from "./DatabaseSchemaUpdateTemplate";

const DEFAULT_TEMPLATE = {
  type: "bytebase.general",
  buildTask: (ctx: TemplateContext): TaskNew => {
    return {
      name: "",
      type: "bytebase.general",
      description: "",
      stageList: [
        {
          id: "1",
          name: "Request",
          type: "bytebase.stage.general",
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
  DatabaseCreateTemplate,
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
