import { fromJson, toJson } from "@bufbuild/protobuf";
import type {
  DatabaseGroup as OldDatabaseGroup,
  DatabaseGroup_Database as OldDatabaseGroup_Database,
  DatabaseGroupView as OldDatabaseGroupView,
} from "@/types/proto/v1/database_group_service";
import {
  DatabaseGroup as OldDatabaseGroupProto,
  DatabaseGroup_Database as OldDatabaseGroup_DatabaseProto,
  DatabaseGroupView as OldDatabaseGroupViewEnum,
} from "@/types/proto/v1/database_group_service";
import type {
  DatabaseGroup as NewDatabaseGroup,
  DatabaseGroup_Database as NewDatabaseGroup_Database,
} from "@/types/proto-es/v1/database_group_service_pb";
import {
  DatabaseGroupSchema,
  DatabaseGroup_DatabaseSchema,
  DatabaseGroupView as NewDatabaseGroupView,
} from "@/types/proto-es/v1/database_group_service_pb";

// Convert old proto DatabaseGroup to proto-es DatabaseGroup
export const convertOldDatabaseGroupToNew = (
  oldGroup: OldDatabaseGroup
): NewDatabaseGroup => {
  const json = OldDatabaseGroupProto.toJSON(oldGroup) as any;
  return fromJson(DatabaseGroupSchema, json);
};

// Convert proto-es DatabaseGroup to old proto DatabaseGroup
export const convertNewDatabaseGroupToOld = (
  newGroup: NewDatabaseGroup
): OldDatabaseGroup => {
  const json = toJson(DatabaseGroupSchema, newGroup);
  return OldDatabaseGroupProto.fromJSON(json);
};

// Convert old DatabaseGroup_Database to new
export const convertOldDatabaseGroup_DatabaseToNew = (
  oldDb: OldDatabaseGroup_Database
): NewDatabaseGroup_Database => {
  const json = OldDatabaseGroup_DatabaseProto.toJSON(oldDb) as any;
  return fromJson(DatabaseGroup_DatabaseSchema, json);
};

// Convert new DatabaseGroup_Database to old
export const convertNewDatabaseGroup_DatabaseToOld = (
  newDb: NewDatabaseGroup_Database
): OldDatabaseGroup_Database => {
  const json = toJson(DatabaseGroup_DatabaseSchema, newDb);
  return OldDatabaseGroup_DatabaseProto.fromJSON(json);
};

// Convert old DatabaseGroupView enum to new
export const convertOldDatabaseGroupViewToNew = (
  oldView: OldDatabaseGroupView
): NewDatabaseGroupView => {
  switch (oldView) {
    case OldDatabaseGroupViewEnum.DATABASE_GROUP_VIEW_UNSPECIFIED:
      return NewDatabaseGroupView.UNSPECIFIED;
    case OldDatabaseGroupViewEnum.DATABASE_GROUP_VIEW_BASIC:
      return NewDatabaseGroupView.BASIC;
    case OldDatabaseGroupViewEnum.DATABASE_GROUP_VIEW_FULL:
      return NewDatabaseGroupView.FULL;
    case OldDatabaseGroupViewEnum.UNRECOGNIZED:
    default:
      return NewDatabaseGroupView.UNSPECIFIED;
  }
};

// Convert new DatabaseGroupView enum to old
export const convertNewDatabaseGroupViewToOld = (
  newView: NewDatabaseGroupView
): OldDatabaseGroupView => {
  switch (newView) {
    case NewDatabaseGroupView.UNSPECIFIED:
      return OldDatabaseGroupViewEnum.DATABASE_GROUP_VIEW_UNSPECIFIED;
    case NewDatabaseGroupView.BASIC:
      return OldDatabaseGroupViewEnum.DATABASE_GROUP_VIEW_BASIC;
    case NewDatabaseGroupView.FULL:
      return OldDatabaseGroupViewEnum.DATABASE_GROUP_VIEW_FULL;
    default:
      return OldDatabaseGroupViewEnum.DATABASE_GROUP_VIEW_UNSPECIFIED;
  }
};