import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Revision as OldRevision } from "@/types/proto/v1/revision_service";
import { Revision as OldRevisionProto } from "@/types/proto/v1/revision_service";
import type { Revision as NewRevision } from "@/types/proto-es/v1/revision_service_pb";
import { RevisionSchema } from "@/types/proto-es/v1/revision_service_pb";

// Convert old proto to proto-es
export const convertOldRevisionToNew = (oldRevision: OldRevision): NewRevision => {
  // Use toJSON to convert old proto to JSON, then fromJson to convert to proto-es
  const json = OldRevisionProto.toJSON(oldRevision) as any; // Type assertion needed due to proto type incompatibility
  return fromJson(RevisionSchema, json);
};

// Convert proto-es to old proto
export const convertNewRevisionToOld = (newRevision: NewRevision): OldRevision => {
  // Use toJson to convert proto-es to JSON, then fromJSON to convert to old proto
  const json = toJson(RevisionSchema, newRevision);
  return OldRevisionProto.fromJSON(json);
};