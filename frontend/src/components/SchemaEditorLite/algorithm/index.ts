import type { SchemaEditorContext } from "../context";
import { useApplyMetadataEdit } from "./apply";
import { useRebuildMetadataEdit } from "./rebuild";

export const useAlgorithm = (context: SchemaEditorContext) => {
  return {
    ...useRebuildMetadataEdit(context),
    ...useApplyMetadataEdit(context),
  };
};
