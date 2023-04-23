import { InstanceId, InstanceUserId } from "./id";

export type InstanceUser = {
  id: InstanceUserId;

  // Related fields
  instanceId: InstanceId;

  // Domain specific fields
  name: string;
  grant: string;
};

export const INTERNAL_RDS_INSTANCE_USER_LIST = [
  "rds_ad",
  "rdsadmin",
  "rds_iam",
];
