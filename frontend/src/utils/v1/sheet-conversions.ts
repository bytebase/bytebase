import { fromJson, toJson } from "@bufbuild/protobuf";
import { Sheet as OldSheet } from "@/types/proto/v1/sheet_service";
import type { Sheet as NewSheet } from "@/types/proto-es/v1/sheet_service_pb";
import { SheetSchema } from "@/types/proto-es/v1/sheet_service_pb";

// Convert old proto to proto-es
export const convertOldSheetToNew = (oldSheet: OldSheet): NewSheet => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldSheet.toJSON(oldSheet) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(SheetSchema, json);
};

// Convert proto-es to old proto
export const convertNewSheetToOld = (newSheet: NewSheet): OldSheet => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(SheetSchema, newSheet);
  return OldSheet.fromJSON(json);
};