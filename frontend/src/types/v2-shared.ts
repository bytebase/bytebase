import type { SelectOption } from "naive-ui";

export type ResourceSelectOption<T> = SelectOption & {
  resource?: T;
  value: string;
  label: string;
};

export type SelectSize = "tiny" | "small" | "medium" | "large";
