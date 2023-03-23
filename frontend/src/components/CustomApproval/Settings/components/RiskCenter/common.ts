import { Risk_Source, risk_SourceToJSON } from "@/types/proto/v1/risk_service";
import { useI18n } from "vue-i18n";

export const sourceText = (source: Risk_Source) => {
  const { t, te } = useI18n();
  const name = risk_SourceToJSON(source);
  const keypath = `custom-approval.security-rule.risk.namespace.${name.toLowerCase()}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return name;
};
