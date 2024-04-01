import type { SQLEditorTreeNode } from "@/types";
import {
  useHoverStateContext as _useHoverStateContext,
  provideHoverStateContext as _provideHoverStateContext,
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
