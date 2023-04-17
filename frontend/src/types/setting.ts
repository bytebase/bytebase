import { SettingId } from "./id";
import {
  SMTPMailDeliverySetting_Encryption,
  SMTPMailDeliverySetting_Authentication,
} from "./proto/store/setting";

export type SettingName =
  | "bb.branding.logo"
  | "bb.app.im"
  | "bb.workspace.watermark"
  | "bb.workspace.profile"
  | "bb.workspace.approval"
  | "bb.plugin.openai.key"
  | "bb.plugin.openai.endpoint"
  | "bb.workspace.mail-delivery";

export type Setting = {
  id: SettingId;

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

export type SMTPAuthenticationType = "NONE" | "PLAIN";
export type SMTPEncryptionType = "NONE" | "SSL/TLS" | "STARTTLS";

export interface SettingWorkspaceMailDeliveryValue {
  smtpServerHost: string;
  smtpServerPort: number;
  smtpFrom: string;
  smtpUsername: string;
  smtpPassword: string | undefined;
  smtpAuthenticationType: SMTPMailDeliverySetting_Authentication;
  smtpEncryptionType: SMTPMailDeliverySetting_Encryption;
}

export interface TestWorkspaceDeliveryValue
  extends SettingWorkspaceMailDeliveryValue {
  sendTo: string;
}
