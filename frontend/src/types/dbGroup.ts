import { create } from "@bufbuild/protobuf";
import { UNKNOWN_ID } from "@/types";
import {
  type DatabaseGroup,
  DatabaseGroupSchema,
} from "@/types/proto-es/v1/database_group_service_pb";
import { extractDatabaseGroupName } from "@/utils";
import { unknownProject } from "./v1/project";

export const isValidDatabaseGroupName = (name: string): boolean => {
  if (typeof name !== "string") return false;
  const dbGroupName = extractDatabaseGroupName(name);
  return Boolean(dbGroupName) && dbGroupName !== String(UNKNOWN_ID);
};

export const unknownDatabaseGroup = (): DatabaseGroup => {
  const projectEntity = unknownProject();
  return create(DatabaseGroupSchema, {
    name: `${projectEntity.name}/databaseGroups/${UNKNOWN_ID}`,
    title: "<<Unknown database group>>",
  });
};
