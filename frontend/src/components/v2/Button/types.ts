import { type ButtonProps } from "naive-ui";

export type ContextMenuButtonAction<T = unknown> = {
  key: string;
  text: string;
  props?: ButtonProps;
  params: T;
};

export type TooltipMode = "ALWAYS" | "DISABLED-ONLY";
