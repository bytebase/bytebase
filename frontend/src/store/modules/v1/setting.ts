import { create } from "@bufbuild/protobuf";
import type { FieldMask } from "@bufbuild/protobuf/wkt";
import { createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { computed, reactive } from "vue";
import { settingServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { settingNamePrefix } from "@/store/modules/v1/common";
import {
  type DataClassificationSetting_DataClassificationConfig,
  GetSettingRequestSchema,
  ListSettingsRequestSchema,
  type Setting,
  Setting_SettingName,
  SettingSchema,
  type SettingValue,
  SettingValueSchema,
  UpdateSettingRequestSchema,
  type WorkspaceProfileSetting,
  WorkspaceProfileSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import { useActuatorV1Store } from "./actuator";

export const useSettingV1Store = defineStore("setting_v1", () => {
  const settingMapByName = reactive(new Map<string, Setting>());

  const workspaceProfile = computed((): WorkspaceProfileSetting => {
    const setting = settingMapByName.get(
      `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.WORKSPACE_PROFILE]}`
    );
    if (setting?.value?.value.case === "workspaceProfile") {
      return setting.value.value.value;
    }
    return create(WorkspaceProfileSettingSchema, {});
  });

  const classification = computed(
    (): DataClassificationSetting_DataClassificationConfig[] => {
      const setting = settingMapByName.get(
        `${settingNamePrefix}${Setting_SettingName[Setting_SettingName.DATA_CLASSIFICATION]}`
      );
      if (!setting?.value?.value) return [];
      const value = setting.value.value;
      if (value.case === "dataClassification") {
        return value.value.configs;
      }
      return [];
    }
  );

  const getProjectClassification = (
    classificationId: string
  ): DataClassificationSetting_DataClassificationConfig | undefined => {
    return classification.value.find(
      (config) => config.id === classificationId
    );
  };

  const fetchSettingByName = async (
    name: Setting_SettingName,
    silent = false
  ): Promise<Setting | undefined> => {
    try {
      const request = create(GetSettingRequestSchema, {
        name: `${settingNamePrefix}${Setting_SettingName[name]}`,
      });
      const response = await settingServiceClientConnect.getSetting(request, {
        contextValues: createContextValues().set(silentContextKey, silent),
      });
      settingMapByName.set(response.name, response);
      return response;
    } catch {
      return;
    }
  };

  const getSettingByName = (name: Setting_SettingName): Setting | undefined => {
    return settingMapByName.get(
      `${settingNamePrefix}${Setting_SettingName[name]}`
    );
  };

  const getOrFetchSettingByName = (
    name: Setting_SettingName,
    silent = false
  ): Promise<Setting | undefined> | Setting | undefined => {
    const setting = getSettingByName(name);
    if (setting) {
      return setting;
    }
    return fetchSettingByName(name, silent);
  };

  const fetchSettingList = async () => {
    if (!hasWorkspacePermissionV2("bb.settings.list")) {
      return;
    }
    const request = create(ListSettingsRequestSchema, {});
    const response = await settingServiceClientConnect.listSettings(request);
    for (const setting of response.settings) {
      settingMapByName.set(setting.name, setting);
    }
  };

  const upsertSetting = async ({
    name,
    value,
    validateOnly = false,
    updateMask,
  }: {
    name: Setting_SettingName;
    value: SettingValue;
    validateOnly?: boolean;
    updateMask?: FieldMask | undefined;
  }): Promise<Setting> => {
    const setting = create(SettingSchema, {
      name: `${settingNamePrefix}${Setting_SettingName[name]}`,
      value,
    });

    const request = create(UpdateSettingRequestSchema, {
      setting,
      validateOnly,
      allowMissing: true,
      updateMask: updateMask,
    });
    const response = await settingServiceClientConnect.updateSetting(request);
    settingMapByName.set(response.name, response);
    return response;
  };

  const updateWorkspaceProfile = async ({
    payload,
    updateMask,
  }: {
    payload: Partial<WorkspaceProfileSetting>;
    updateMask: FieldMask;
  }): Promise<void> => {
    const profileSetting: WorkspaceProfileSetting = create(
      WorkspaceProfileSettingSchema,
      {
        ...workspaceProfile.value,
        ...payload,
      }
    );
    await upsertSetting({
      name: Setting_SettingName.WORKSPACE_PROFILE,
      value: create(SettingValueSchema, {
        value: {
          case: "workspaceProfile",
          value: profileSetting,
        },
      }),
      updateMask: updateMask,
    });
    // Refresh the latest server info.
    await useActuatorV1Store().fetchServerInfo();
  };

  return {
    settingMapByName,
    workspaceProfile,
    classification,
    getProjectClassification,
    fetchSettingByName,
    getSettingByName,
    getOrFetchSettingByName,
    fetchSettingList,
    upsertSetting,
    updateWorkspaceProfile,
  };
});

export const useSettingByName = (name: Setting_SettingName) => {
  const store = useSettingV1Store();
  const setting = computed(() => store.getSettingByName(name));
  store.getOrFetchSettingByName(name, /* silent */ true);
  return setting;
};
