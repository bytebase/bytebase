import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { RulesSection } from "@/react/components/CustomApproval/RulesSection";
import { APPROVAL_SOURCES } from "@/react/components/CustomApproval/utils";
import { FeatureAttention } from "@/react/components/FeatureAttention";
import { useVueState } from "@/react/hooks/useVueState";
import {
  useSubscriptionV1Store,
  useWorkspaceApprovalSettingStore,
} from "@/store";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

export function CustomApprovalPage() {
  const { t } = useTranslation();
  const store = useWorkspaceApprovalSettingStore();
  const subscriptionStore = useSubscriptionV1Store();
  const featureAttentionRef = useRef<HTMLDivElement>(null);

  const [ready, setReady] = useState(false);

  const hasFeature = useVueState(() =>
    subscriptionStore.hasInstanceFeature(PlanFeature.FEATURE_APPROVAL_WORKFLOW)
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
        <div className="textinfolabel">
          {t("custom-approval.rule.first-match-wins")}
        </div>
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
  store: ReturnType<typeof useWorkspaceApprovalSettingStore>;
  allowAdmin: boolean;
  hasFeature: boolean;
  onShowFeatureModal: () => void;
}) {
  const rules = useVueState(() => store.getRulesBySource(source));

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
