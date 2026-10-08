import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { AccountType, getAccountTypeByEmail } from "@/types/v1/user";

/** Marks a service account or a workload identity; nothing for a person. */
export function AccountTypeBadge({ email }: { email: string | undefined }) {
  const type = email ? getAccountTypeByEmail(email) : AccountType.USER;
  return type === AccountType.USER ? null : <AccountTypeLabel type={type} />;
}

function AccountTypeLabel({ type }: { type: AccountType }) {
  const { t } = useTranslation();
  return (
    <Badge variant="secondary" className="text-xs">
      {type === AccountType.SERVICE_ACCOUNT
        ? t("settings.members.service-account")
        : t("settings.members.workload-identity")}
    </Badge>
  );
}
