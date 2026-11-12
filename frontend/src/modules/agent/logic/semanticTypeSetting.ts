import { create } from "@bufbuild/protobuf";
import { Code, ConnectError, createContextValues } from "@connectrpc/connect";
import { settingServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import {
  GetSettingRequestSchema,
  type Setting,
  SettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";

export async function readSemanticTypeSetting(): Promise<{
  setting: Setting;
  configured: boolean;
}> {
  const name = "settings/SEMANTIC_TYPES";
  try {
    const setting = await settingServiceClientConnect.getSetting(
      create(GetSettingRequestSchema, { name }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
    return { setting, configured: true };
  } catch (error) {
    if (ConnectError.from(error).code !== Code.NotFound) throw error;
    return {
      setting: create(SettingSchema, {
        name,
        value: { value: { case: "semanticType", value: { types: [] } } },
      }),
      configured: false,
    };
  }
}
