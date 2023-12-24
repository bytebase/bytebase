import { SchemaEditorContext } from "../context";
import { useApplyMetadataEdit } from "./apply";
import { useApplySelectedMetadataEdit } from "./apply-selected";
import { useRebuildMetadataEdit } from "./rebuild";

export const useAlgorithm = (context: SchemaEditorContext) => {
  return {
    ...useRebuildMetadataEdit(context),
    ...useApplyMetadataEdit(context),
    ...useApplySelectedMetadataEdit(context),
  };
};
