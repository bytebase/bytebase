import { t } from "@/plugins/i18n";
import {
  SemanticTypeSetting_SemanticType,
  Algorithm,
} from "@/types/proto/api/v1alpha/setting_service";
import buildInSemanticTypes from "./semantic-types.yaml";

interface BuildInSemantic {
  id: string;
  algorithm: Algorithm;
}

export const getSemanticTemplateList = () => {
  return (buildInSemanticTypes as BuildInSemantic[]).map((buildInSemantic) => {
    const key = buildInSemantic.id.split(".").join("-");
    return SemanticTypeSetting_SemanticType.fromPartial({
      id: buildInSemantic.id,
      title: t(`dynamic.settings.sensitive-data.semantic-types.template.${key}.title`),
      description: t(
        `dynamic.settings.sensitive-data.semantic-types.template.${key}.description`
      ),
      algorithm: buildInSemantic.algorithm,
    });
  });
};
