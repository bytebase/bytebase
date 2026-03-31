import { useTranslation } from "react-i18next";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { Badge } from "@/react/components/ui/badge";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";

export function RiskCenterPage() {
  const { t } = useTranslation();

  return (
    <div className="w-full px-4 py-4 flex flex-col gap-y-6 text-sm">
      <FeatureAttention feature={PlanFeature.FEATURE_RISK_ASSESSMENT} />

      <div className="textinfolabel">
        {t("custom-approval.risk.description")}
      </div>

      <div className="space-y-6">
        <div>
          <h3 className="text-lg font-medium text-main mb-2">
            {t("custom-approval.risk.how-it-works")}
          </h3>
          <p className="text-control-light">
            {t("custom-approval.risk.how-it-works-description")}
          </p>
        </div>

        <div>
          <h3 className="text-lg font-medium text-main mb-3">
            {t("custom-approval.risk.risk-levels")}
          </h3>
          <div className="space-y-3">
            <div className="flex items-start gap-x-3">
              <Badge variant="destructive">
                {t("custom-approval.risk-rule.risk.risk-level.high")}
              </Badge>
              <p className="text-control-light">
                {t("custom-approval.risk.risk-level-high")}
              </p>
            </div>
            <div className="flex items-start gap-x-3">
              <Badge variant="warning">
                {t("custom-approval.risk-rule.risk.risk-level.moderate")}
              </Badge>
              <p className="text-control-light">
                {t("custom-approval.risk.risk-level-moderate")}
              </p>
            </div>
            <div className="flex items-start gap-x-3">
              <Badge variant="secondary">
                {t("custom-approval.risk-rule.risk.risk-level.low")}
              </Badge>
              <p className="text-control-light">
                {t("custom-approval.risk.risk-level-low")}
              </p>
            </div>
          </div>
        </div>

        <div>
          <h3 className="text-lg font-medium text-main mb-2">
            {t("custom-approval.risk.integration")}
          </h3>
          <p className="text-control-light">
            {t("custom-approval.risk.integration-description")}
          </p>
        </div>
      </div>
    </div>
  );
}
