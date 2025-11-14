import type { SQLEditorTreeNode } from "@/types";
import {
  provideHoverStateContext as _provideHoverStateContext,
  useHoverStateContext as _useHoverStateContext,
} from "../../../EditorCommon";

export const KEY = "connection-pane";

export type HoverState = {
  node?: SQLEditorTreeNode;
};

export const useHoverStateContext = () => {
  return _useHoverStateContext<HoverState>(KEY);
};

export const provideHoverStateContext = () => {
  return _provideHoverStateContext<HoverState>(KEY);
};
