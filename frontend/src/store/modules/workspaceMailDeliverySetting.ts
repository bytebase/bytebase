import { settingServiceClient } from "@/grpcweb";
import { Setting } from "@/types/proto/v1/setting_service";
import { SMTPMailDeliverySetting } from "@/types/proto/store/setting";
import { defineStore } from "pinia";
import { ref } from "vue";

const SETTING_NAME = "settings/bb.workspace.mail-delivery";

export const useWorkspaceMailDeliverySettingStore = defineStore(
  "workspaceMailDeliverySetting",
  () => {
    const mailDeliverySetting = ref<SMTPMailDeliverySetting>();

    const setMailDeliverySetting = (setting: Setting) => {
      const _mailDeliverySetting = setting.value!.smtpMailDeliverySettingValue;
      if (_mailDeliverySetting) {
        mailDeliverySetting.value = _mailDeliverySetting;
      }
    };

    const fetchMailDeliverySetting = async () => {
      try {
        const setting = await settingServiceClient.getSetting({
          name: SETTING_NAME,
        });
        setMailDeliverySetting(setting);
      } catch (ex) {
        console.error(ex);
      }
    };

    const updateMailDeliverySetting = async (
      value: SMTPMailDeliverySetting
    ) => {
      const setting = await settingServiceClient.setSetting({
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
      value: SMTPMailDeliverySetting
    ) => {
      await settingServiceClient.setSetting({
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
