import { defineStore } from "pinia";
import { ref } from "vue";
import { settingServiceClient } from "@/grpcweb";
import type {
  SMTPMailDeliverySettingValue,
  Setting,
} from "@/types/proto/api/v1alpha/setting_service";

const SETTING_NAME = "settings/bb.workspace.mail-delivery";

export const useWorkspaceMailDeliverySettingStore = defineStore(
  "workspaceMailDeliverySetting",
  () => {
    const mailDeliverySetting = ref<SMTPMailDeliverySettingValue>();

    const setMailDeliverySetting = (setting: Setting) => {
      const _mailDeliverySetting = setting.value!.smtpMailDeliverySettingValue;
      if (_mailDeliverySetting) {
        mailDeliverySetting.value = _mailDeliverySetting;
      }
    };

    const fetchMailDeliverySetting = async () => {
      const setting = await settingServiceClient.getSetting(
        {
          name: SETTING_NAME,
        },
        {
          silent: true,
        }
      );
      setMailDeliverySetting(setting);
    };

    const updateMailDeliverySetting = async (
      value: SMTPMailDeliverySettingValue
    ) => {
      const setting = await settingServiceClient.updateSetting({
        allowMissing: true,
        setting: {
          name: SETTING_NAME,
          value: {
            smtpMailDeliverySettingValue: value,
          },
        },
      });
      setMailDeliverySetting(setting);
    };

    const validateMailDeliverySetting = async (
      value: SMTPMailDeliverySettingValue
    ) => {
      await settingServiceClient.updateSetting({
        allowMissing: true,
        setting: {
          name: SETTING_NAME,
          value: {
            smtpMailDeliverySettingValue: value,
          },
        },
        validateOnly: true,
      });
    };

    return {
      mailDeliverySetting,
      fetchMailDeliverySetting,
      updateMailDeliverySetting,
      validateMailDeliverySetting,
    };
  }
);
