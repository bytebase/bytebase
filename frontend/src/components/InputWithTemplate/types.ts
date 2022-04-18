export interface Template {
  id: string;
  description?: string;
}

export enum InputType {
  String = "string",
  Template = "template",
}

export interface TemplateInput {
  value: string;
  type: InputType;
}
