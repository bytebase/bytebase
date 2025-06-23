import { fromJson, toJson } from "@bufbuild/protobuf";
import type { Changelist as OldChangelist } from "@/types/proto/v1/changelist_service";
import { Changelist as OldChangelistProto } from "@/types/proto/v1/changelist_service";
import type { Changelist as NewChangelist } from "@/types/proto-es/v1/changelist_service_pb";
import { ChangelistSchema } from "@/types/proto-es/v1/changelist_service_pb";

// Convert old proto to proto-es
export const convertOldChangelistToNew = (oldChangelist: OldChangelist): NewChangelist => {
  const json = OldChangelistProto.toJSON(oldChangelist) as any;
  return fromJson(ChangelistSchema, json);
};

// Convert proto-es to old proto
export const convertNewChangelistToOld = (newChangelist: NewChangelist): OldChangelist => {
  const json = toJson(ChangelistSchema, newChangelist);
  return OldChangelistProto.fromJSON(json);
};