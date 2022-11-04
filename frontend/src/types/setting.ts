import { SettingId } from "./id";
import { Principal } from "./principal";

export type SettingName = "bb.branding.logo" | "bb.app.im";

export type Setting = {
  id: SettingId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: SettingName;
  value: string;
  description: string;
};

type IMType = "im.feishu";

export interface SettingAppIMValue {
  imType: IMType;
  appId: string;
  appSecret: string;
  externalApproval: {
    enabled: boolean;
  };
}
