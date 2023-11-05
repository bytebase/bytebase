import { v4 as uuidv4 } from "uuid";
import { useI18n } from "vue-i18n";
import { SemanticTypeSetting_SemanticType } from "@/types/proto/v1/setting_service";
import buildInSemanticTypes from "./semantic-types.yaml";

interface BuildInSemantic {
  id: string;
  fullMaskAlgorithmId?: string;
  partialMaskAlgorithmId?: string;
}

export const getSemanticTemplateList = () => {
  const { t } = useI18n();
  return (buildInSemanticTypes as BuildInSemantic[]).map((buildInSemantic) =>
    SemanticTypeSetting_SemanticType.fromPartial({
      id: uuidv4(),
      title: t(
        `settings.sensitive-data.semantic-types.template.${buildInSemantic.id}.title`
      ),
      description: "",
      fullMaskAlgorithmId: buildInSemantic.fullMaskAlgorithmId,
      partialMaskAlgorithmId: buildInSemantic.partialMaskAlgorithmId,
    })
  );
};
