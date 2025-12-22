import { create } from "@bufbuild/protobuf";
import { t } from "@/plugins/i18n";
import type { Algorithm } from "@/types/proto-es/v1/setting_service_pb";
import {
  type SemanticTypeSetting_SemanticType,
  SemanticTypeSetting_SemanticTypeSchema,
} from "@/types/proto-es/v1/setting_service_pb";
import buildInSemanticTypes from "./semantic-types.yaml";

interface BuildInSemantic {
  id: string;
  algorithm: Algorithm;
}

export const getSemanticTemplateList =
  (): SemanticTypeSetting_SemanticType[] => {
    return (buildInSemanticTypes as unknown as BuildInSemantic[]).map(
      (buildInSemantic) => {
        const key = buildInSemantic.id.split(".").join("-");
        return create(SemanticTypeSetting_SemanticTypeSchema, {
          id: buildInSemantic.id,
          title: t(
            `dynamic.settings.sensitive-data.semantic-types.template.${key}.title`
          ),
          description: t(
            `dynamic.settings.sensitive-data.semantic-types.template.${key}.description`
          ),
          algorithm: buildInSemantic.algorithm,
        });
      }
    );
  };
