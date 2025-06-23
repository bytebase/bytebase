import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Release as OldRelease, CheckReleaseResponse as OldCheckReleaseResponse } from "@/types/proto/v1/release_service";
import { Release as OldReleaseProto, CheckReleaseResponse as OldCheckReleaseResponseProto, Release_File_ChangeType as OldChangeType } from "@/types/proto/v1/release_service";
import type { Release as NewRelease, CheckReleaseResponse as NewCheckReleaseResponse } from "@/types/proto-es/v1/release_service_pb";
import { ReleaseSchema, CheckReleaseResponseSchema, Release_File_ChangeType as NewChangeType } from "@/types/proto-es/v1/release_service_pb";

// Convert old proto to proto-es
export const convertOldReleaseToNew = (oldRelease: OldRelease): NewRelease => {
  const json = OldReleaseProto.toJSON(oldRelease) as any;
  return fromJson(ReleaseSchema, json);
};

// Convert proto-es to old proto
export const convertNewReleaseToOld = (newRelease: NewRelease): OldRelease => {
  const json = toJson(ReleaseSchema, newRelease);
  return OldReleaseProto.fromJSON(json);
};

// Convert proto-es CheckReleaseResponse to old proto
export const convertNewCheckReleaseResponseToOld = (newResponse: NewCheckReleaseResponse): OldCheckReleaseResponse => {
  const json = toJson(CheckReleaseResponseSchema, newResponse);
  return OldCheckReleaseResponseProto.fromJSON(json);
};

// Convert old ChangeType enum to new ChangeType enum
export const convertOldChangeTypeToNew = (oldChangeType: OldChangeType): NewChangeType => {
  switch (oldChangeType) {
    case OldChangeType.DDL:
      return NewChangeType.DDL;
    case OldChangeType.DDL_GHOST:
      return NewChangeType.DDL_GHOST;
    case OldChangeType.DML:
      return NewChangeType.DML;
    default:
      return NewChangeType.CHANGE_TYPE_UNSPECIFIED;
  }
};