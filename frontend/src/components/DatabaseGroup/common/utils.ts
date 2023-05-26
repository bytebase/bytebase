import type { Factor } from "@/plugins/cel";
import { t, te } from "@/plugins/i18n";

const stringifyFactor = (factor: Factor) => {
  return factor.replace(/\./g, "_");
};

export const factorText = (factor: Factor) => {
  const keypath = `database-group.factor.${stringifyFactor(factor)}`;
  if (te(keypath)) {
    return t(keypath);
  }
  return factor;
};
