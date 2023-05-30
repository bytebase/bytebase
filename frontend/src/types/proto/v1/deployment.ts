/* eslint-disable */

export const protobufPackage = "bytebase.v1";

export enum DeploymentType {
  DEPLOYMENT_TYPE_UNSPECIFIED = 0,
  DATABASE_CREATE = 1,
  DATABASE_DDL = 2,
  DATABASE_DDL_GHOST = 3,
  DATABASE_DML = 4,
  DATABASE_RESTORE_PITR = 5,
  UNRECOGNIZED = -1,
}

export function deploymentTypeFromJSON(object: any): DeploymentType {
  switch (object) {
    case 0:
    case "DEPLOYMENT_TYPE_UNSPECIFIED":
      return DeploymentType.DEPLOYMENT_TYPE_UNSPECIFIED;
    case 1:
    case "DATABASE_CREATE":
      return DeploymentType.DATABASE_CREATE;
    case 2:
    case "DATABASE_DDL":
      return DeploymentType.DATABASE_DDL;
    case 3:
    case "DATABASE_DDL_GHOST":
      return DeploymentType.DATABASE_DDL_GHOST;
    case 4:
    case "DATABASE_DML":
      return DeploymentType.DATABASE_DML;
    case 5:
    case "DATABASE_RESTORE_PITR":
      return DeploymentType.DATABASE_RESTORE_PITR;
    case -1:
    case "UNRECOGNIZED":
    default:
      return DeploymentType.UNRECOGNIZED;
  }
}

export function deploymentTypeToJSON(object: DeploymentType): string {
  switch (object) {
    case DeploymentType.DEPLOYMENT_TYPE_UNSPECIFIED:
      return "DEPLOYMENT_TYPE_UNSPECIFIED";
    case DeploymentType.DATABASE_CREATE:
      return "DATABASE_CREATE";
    case DeploymentType.DATABASE_DDL:
      return "DATABASE_DDL";
    case DeploymentType.DATABASE_DDL_GHOST:
      return "DATABASE_DDL_GHOST";
    case DeploymentType.DATABASE_DML:
      return "DATABASE_DML";
    case DeploymentType.DATABASE_RESTORE_PITR:
      return "DATABASE_RESTORE_PITR";
    case DeploymentType.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
