import {
  useHoverStateContext as _useHoverStateContext,
  provideHoverStateContext as _provideHoverStateContext,
} from "../../../EditorCommon";

export const KEY = "schema-pane";

export type HoverState = {
  database: string;
  schema?: string;
  table?: string;
  externalTable?: string;
  view?: string;
  column?: string;
  partition?: string;
};

export const useHoverStateContext = () => {
  return _useHoverStateContext<HoverState>(KEY);
};

export const provideHoverStateContext = () => {
  return _provideHoverStateContext<HoverState>(KEY);
};
