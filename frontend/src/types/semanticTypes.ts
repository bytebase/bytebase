import { create } from "@bufbuild/protobuf";
import i18n from "@/lib/i18n";
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

const builtinSemanticTypeIds = new Set(
  (buildInSemanticTypes as unknown as BuildInSemantic[]).map(({ id }) => id)
);

export const isBuiltinSemanticTypeId = (id: string): boolean =>
  builtinSemanticTypeIds.has(id);

export const getSemanticTemplateList =
  (): SemanticTypeSetting_SemanticType[] => {
    return (buildInSemanticTypes as unknown as BuildInSemantic[]).map(
      (buildInSemantic) => {
        const key = buildInSemantic.id.split(".").join("-");
        return create(SemanticTypeSetting_SemanticTypeSchema, {
          id: buildInSemantic.id,
          title: i18n.t(
            `dynamic.settings.sensitive-data.semantic-types.template.${key}.title`
          ),
          description: i18n.t(
            `dynamic.settings.sensitive-data.semantic-types.template.${key}.description`
          ),
          algorithm: buildInSemantic.algorithm,
        });
      }
    );
  };

export const getSemanticTypeListWithBuiltins = (
  semanticTypeList: SemanticTypeSetting_SemanticType[]
): SemanticTypeSetting_SemanticType[] => {
  const builtins = getSemanticTemplateList();
  return [
    ...builtins,
    ...semanticTypeList.filter(
      (semanticType) => !isBuiltinSemanticTypeId(semanticType.id)
    ),
  ];
};
