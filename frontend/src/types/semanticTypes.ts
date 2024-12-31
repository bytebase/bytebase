import { v4 as uuidv4 } from "uuid";
import { t } from "@/plugins/i18n";
import { SemanticTypeSetting_SemanticType, Algorithm } from "@/types/proto/v1/setting_service";
import buildInSemanticTypes from "./semantic-types.yaml";

interface BuildInSemantic {
  id: string;
  algorithm: Algorithm;
}

export const getSemanticTemplateList = () => {
  return (buildInSemanticTypes as BuildInSemantic[]).map((buildInSemantic) =>
    SemanticTypeSetting_SemanticType.fromPartial({
      id: uuidv4(),
      title: t(
        `settings.sensitive-data.semantic-types.template.${buildInSemantic.id}.title`
      ),
      description: t(
        `settings.sensitive-data.semantic-types.template.${buildInSemantic.id}.description`
      ),
      algorithms: buildInSemantic.algorithm,
    })
  );
};
