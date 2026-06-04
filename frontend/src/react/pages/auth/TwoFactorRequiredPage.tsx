import { useTranslation } from "react-i18next";
import { Alert } from "@/react/components/ui/alert";
import { TwoFactorSetupPage } from "@/react/pages/settings/two-factor/TwoFactorSetupPage";
import { useAppStore } from "@/react/stores/app";

export function TwoFactorRequiredPage() {
  const { t } = useTranslation();
  return (
    <div className="w-full">
      <Alert
        variant="warning"
        description={t("two-factor.messages.2fa-required")}
      />
      <div className="w-full p-2 sm:p-8 sm:px-16">
        <TwoFactorSetupPage
          cancelAction={() => useAppStore.getState().logout()}
        />
      </div>
    </div>
  );
}
