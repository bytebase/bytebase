import { QueryClient, useMutation, useQuery } from "@tanstack/vue-query";
import { defineStore } from "pinia";
import { settingServiceClient } from "@/grpcweb";
import { settingNamePrefix } from "@/store/modules/v1/common";
import {
  Setting,
  Value as SettingValue,
  WorkspaceProfileSetting,
  DataClassificationSetting_DataClassificationConfig,
} from "@/types/proto/v1/setting_service";
import { SettingName } from "@/types/setting";
import { useActuatorV1Store } from "./actuator";

interface SettingState {
  settingMapByName: Map<string, Setting>;
}

export const useSettingV1Store = defineStore("setting_v1", {
  state: (): SettingState => ({
    settingMapByName: new Map(),
  }),
  getters: {
    workspaceProfileSetting(state): WorkspaceProfileSetting | undefined {
      const setting = state.settingMapByName.get(
        `${settingNamePrefix}bb.workspace.profile`
      );
      return setting?.value?.workspaceProfileSettingValue;
    },
    brandingLogo(state): string | undefined {
      const setting = state.settingMapByName.get(
        `${settingNamePrefix}bb.branding.logo`
      );
      return setting?.value?.stringValue;
    },
    classification(
      state
    ): DataClassificationSetting_DataClassificationConfig[] {
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}bb.workspace.data-classification`
      );
      return setting?.value?.dataClassificationSettingValue?.configs ?? [];
    },
  },
  actions: {
    getProjectClassification(
      classificationId: string
    ): DataClassificationSetting_DataClassificationConfig | undefined {
      const setting = this.settingMapByName.get(
        `${settingNamePrefix}bb.workspace.data-classification`
      );
      return setting?.value?.dataClassificationSettingValue?.configs.find(
        (config) => config.id === classificationId
      );
    },
    async fetchSettingByName(name: SettingName, silent = false) {
      try {
        const setting = await settingServiceClient.getSetting(
          {
            name: `${settingNamePrefix}${name}`,
          },
          { silent }
        );
        this.settingMapByName.set(setting.name, setting);
        return setting;
      } catch {
        return;
      }
    },
    getOrFetchSettingByName(name: SettingName, silent = false) {
      const setting = this.getSettingByName(name);
      if (setting) {
        return setting;
      }
      return this.fetchSettingByName(name, silent);
    },
    getSettingByName(name: SettingName) {
      return this.settingMapByName.get(`${settingNamePrefix}${name}`);
    },
    async fetchSettingList() {
      const { settings } = await settingServiceClient.listSettings({});
      for (const setting of settings) {
        this.settingMapByName.set(setting.name, setting);
      }
    },
    async upsertSetting({
      name,
      value,
      validateOnly = false,
    }: {
      name: SettingName;
      value: SettingValue;
      validateOnly?: boolean;
    }) {
      const resp = await settingServiceClient.setSetting({
        setting: {
          name: `${settingNamePrefix}${name}`,
          value,
        },
        validateOnly,
      });
      this.settingMapByName.set(resp.name, resp);
      return resp;
    },
    async updateWorkspaceProfile(
      payload: Partial<WorkspaceProfileSetting>
    ): Promise<void> {
      if (!this.workspaceProfileSetting) {
        return;
      }
      const profileSetting: WorkspaceProfileSetting = {
        ...this.workspaceProfileSetting,
        ...payload,
      };
      await this.upsertSetting({
        name: "bb.workspace.profile",
        value: {
          workspaceProfileSettingValue: profileSetting,
        },
      });
      // Fetch the latest server info to refresh the disallow signup flag.
      await useActuatorV1Store().fetchServerInfo();
    },
  },
});

export const useSettingSWRStore = defineStore("setting_swr", () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 5 * 60 * 1000, // 5 minutes
      },
    },
  });
  const useSettingByName = (name: SettingName, silent = false) => {
    const query = useQuery({
      queryClient,
      queryKey: [name],
      queryFn: async () => {
        return await settingServiceClient.getSetting(
          {
            name: `${settingNamePrefix}${name}`,
          },
          { silent }
        );
      },
    });
    return query;
  };
  const useUpdateSettingByName = (name: SettingName) => {
    const mutation = useMutation({
      queryClient,
      mutationFn: async (params: {
        value: SettingValue;
        validateOnly?: boolean;
      }) => {
        const { value, validateOnly } = params;
        const resp = await settingServiceClient.setSetting({
          setting: {
            name: `${settingNamePrefix}${name}`,
            value,
          },
          validateOnly,
        });
        return resp.value;
      },
      onSuccess: (value) => {
        queryClient.setQueryData(
          [name],
          Setting.fromPartial({
            name,
            value,
          })
        );
      },
    });
    return mutation;
  };

  return {
    queryClient,
    useSettingByName,
    useUpdateSettingByName,
  };
});
