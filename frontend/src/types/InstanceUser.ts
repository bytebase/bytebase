import { InstanceId, InstanceUserId } from "./id";

export type InstanceUser = {
  id: InstanceUserId;

  // Related fields
  instanceId: InstanceId;

  // Domain specific fields
  name: string;
  grant: string;
};
