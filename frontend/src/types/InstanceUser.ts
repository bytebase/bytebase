import { InstanceID, InstanceUserID } from "./id";

export type InstanceUser = {
  id: InstanceUserID;

  // Related fields
  instanceID: InstanceID;

  // Domain specific fields
  name: string;
  grant: string;
};
