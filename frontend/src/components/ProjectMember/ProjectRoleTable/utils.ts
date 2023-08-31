import { useI18n } from "vue-i18n";
import { Binding } from "@/types/proto/v1/iam_policy";
import { convertFromExpr } from "@/utils/issue/cel";

export const getExpiredTimeString = (binding: Binding) => {
  const { t } = useI18n();
  const expiredDateTime = getExpiredDateTime(binding);
  return (
    expiredDateTime?.toLocaleString() ?? t("project.members.never-expires")
  );
};

export const isExpired = (binding: Binding) => {
  const expiredDateTime = getExpiredDateTime(binding);
  if (!expiredDateTime) {
    return false;
  }
  return new Date().getTime() >= expiredDateTime.getTime();
};

export const getExpiredDateTime = (binding: Binding) => {
  const parsedExpr = binding.parsedExpr;
  if (parsedExpr?.expr) {
    const expression = convertFromExpr(parsedExpr.expr);
    if (expression.expiredTime) {
      return new Date(expression.expiredTime);
    }
  }
};
