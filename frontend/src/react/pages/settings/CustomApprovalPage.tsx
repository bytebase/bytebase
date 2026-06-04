import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { RulesSection } from "@/react/components/CustomApproval/RulesSection";
import { APPROVAL_SOURCES } from "@/react/components/CustomApproval/utils";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { Alert } from "@/react/components/ui/alert";
import { useAppStore } from "@/react/stores/app";
import {
  useWorkspaceApprovalSettingStore,
  type WorkspaceApprovalSettingState,
} from "@/react/stores/workspaceApprovalSetting";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export function CustomApprovalPage() {
  const { t } = useTranslation();
  const store = useWorkspaceApprovalSettingStore();
  const featureAttentionRef = useRef<HTMLDivElement>(null);

  const [ready, setReady] = useState(false);

  const hasFeature = useAppStore((s) =>
    s.hasInstanceFeature(PlanFeature.FEATURE_APPROVAL_WORKFLOW)
  );
  const allowAdmin = hasWorkspacePermissionV2("bb.settings.set");

  const handleShowFeatureModal = useCallback(() => {
    featureAttentionRef.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  useEffect(() => {
    store
      .fetchConfig()
      .then(() => setReady(true))
      .catch(() => setReady(true));
  }, []);

  return (
    <div className="w-full px-4 py-4 flex flex-col gap-y-4 text-sm">
      <div ref={featureAttentionRef}>
        <FeatureAttention feature={PlanFeature.FEATURE_APPROVAL_WORKFLOW} />
      </div>

      {hasFeature && (
        <Alert variant="info">
          <ul className="flex flex-col gap-y-1 list-disc pl-5">
            <li>
              {t(
                "custom-approval.rule.approval-rule-matching.evaluation-order"
              )}
            </li>
            <li>
              {t("custom-approval.rule.approval-rule-matching.multi-target")}
            </li>
            <li>{t("custom-approval.rule.approval-rule-matching.priority")}</li>
          </ul>
        </Alert>
      )}

      {!ready ? (
        <div className="flex justify-center py-8">
          <Loader2 className="animate-spin" />
        </div>
      ) : (
        <div className="flex flex-col gap-y-6">
          {APPROVAL_SOURCES.map((source) => (
            <SourceSection
              key={source}
              source={source}
              store={store}
              allowAdmin={allowAdmin}
              hasFeature={hasFeature}
              onShowFeatureModal={handleShowFeatureModal}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function SourceSection({
  source,
  store,
  allowAdmin,
  hasFeature,
  onShowFeatureModal,
}: {
  source: (typeof APPROVAL_SOURCES)[number];
  store: WorkspaceApprovalSettingState;
  allowAdmin: boolean;
  hasFeature: boolean;
  onShowFeatureModal: () => void;
}) {
  // getRulesBySource returns a fresh filtered array each call, so memoize on
  // the store + source to keep a stable reference.
  const rules = useMemo(() => store.getRulesBySource(source), [store, source]);

  return (
    <RulesSection
      source={source}
      rules={rules}
      allowAdmin={allowAdmin}
      hasFeature={hasFeature}
      onShowFeatureModal={onShowFeatureModal}
    />
  );
}
