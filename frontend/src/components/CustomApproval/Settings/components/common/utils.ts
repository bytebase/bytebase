import type { Factor } from "@/plugins/cel";
import { t, te } from "@/plugins/i18n";
import {
  Risk,
  Risk_Source,
  risk_SourceToJSON,
} from "@/types/proto/v1/risk_service";

export const sourceText = (source: Risk_Source) => {
  if (source === Risk_Source.SOURCE_UNSPECIFIED) {
    return t("common.all");
  }

  const name = risk_SourceToJSON(source);
  const keypath = `custom-approval.security-rule.risk.namespace.${name.toLowerCase()}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return name;
};

export const levelText = (level: number) => {
  const keypath = `custom-approval.security-rule.risk.risk-level.${level}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return String(level);
};

export const orderByLevelDesc = (a: Risk, b: Risk): number => {
  if (a.level !== b.level) return -(a.level - b.level);
  if (a.name === b.name) return 0;
  return a.name < b.name ? -1 : 1;
};

export const factorText = (factor: Factor) => {
  const keypath = `custom-approval.security-rule.risk.factor.${factor}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return factor;
};
