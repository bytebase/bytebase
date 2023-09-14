import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import {
  BranchSchema,
  convertSchemaMetadataList,
} from "@/types/v1/schemaEditor";
import { rebuildEditableSchemas } from "./metadataV1";

export const convertBranchToBranchSchema = (
  branch: SchemaDesign
): BranchSchema => {
  const originalSchemas = convertSchemaMetadataList(
    branch.baselineSchemaMetadata?.schemas || []
  );
  const editableSchemas = rebuildEditableSchemas(
    originalSchemas,
    branch.schemaMetadata?.schemas || []
  );
  return {
    branch,
    schemaList: editableSchemas,
    originSchemaList: originalSchemas,
  };
};
