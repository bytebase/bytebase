import { groupNamePrefix } from "@/store/modules/v1/common";
import { UNKNOWN_ID } from "../const";
import { GroupSchema } from "../proto-es/v1/group_service_pb";
import { create as createProto } from "@bufbuild/protobuf";

export const UNKNOWN_GROUP_NAME = `${groupNamePrefix}${UNKNOWN_ID}`;

export const unknownGroup = () => {
  return createProto(GroupSchema, {
    name: UNKNOWN_GROUP_NAME,
    title: "<<Unknown group>>",
  });
};
