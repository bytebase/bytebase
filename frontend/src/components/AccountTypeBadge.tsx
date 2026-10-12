import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { AccountType, getAccountTypeByEmail } from "@/types/v1/user";

/**
 * Marks a service account or a workload identity; nothing for a person.
 * `compact` fits a one-line row beside a name that truncates.
 */
export function AccountTypeBadge({
  compact = false,
  email,
}: {
  compact?: boolean;
  email: string | undefined;
}) {
  const type = email ? getAccountTypeByEmail(email) : AccountType.USER;
  return type === AccountType.USER ? null : (
    <AccountTypeLabel compact={compact} type={type} />
  );
}

function AccountTypeLabel({
  compact,
  type,
}: {
  compact: boolean;
  type: AccountType;
}) {
  const { t } = useTranslation();
  return (
    <Badge
      variant="secondary"
      className={cn(
        "text-xs",
        compact && "shrink-0 whitespace-nowrap px-1.5 py-0"
      )}
    >
      {type === AccountType.SERVICE_ACCOUNT
        ? t("settings.members.service-account")
        : t("settings.members.workload-identity")}
    </Badge>
  );
}
