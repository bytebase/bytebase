import { create } from "@bufbuild/protobuf";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { STANDARD_RULE_TYPES } from "@/components/sql-review/standardRules";
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { FormSection } from "@/components/ui/form";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { useWorkspaceResourceName } from "@/hooks/useAppState";
import { useAppStore } from "@/stores/app";
import {
  PolicyResourceType,
  PolicyType,
  ReviewRulePolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import {
  devSqlReviewVersion,
  type SqlReviewVersion,
  setDevSqlReviewVersion,
} from "@/utils/featureGates";

const VERSION_OPTIONS = [
  { value: "v1" as const, label: "V1" },
  { value: "v2" as const, label: "V2" },
];

// Dev builds only. The server runs V1's SQL review for a project whose
// standard rules are all off, so switching writes the workspace standard rules
// as well as this browser's choice of pages. The page then reloads, since the
// SQL review gate is read outside React state.
export function SQLReviewVersionSwitch() {
  const { t } = useTranslation();
  const workspace = useWorkspaceResourceName();
  const current = devSqlReviewVersion();
  const [target, setTarget] = useState<SqlReviewVersion>();
  const [switching, setSwitching] = useState(false);

  const switchVersion = async (version: SqlReviewVersion) => {
    setSwitching(true);
    try {
      await useAppStore.getState().upsertPolicy({
        parentPath: workspace,
        policy: {
          type: PolicyType.REVIEW_RULE,
          resourceType: PolicyResourceType.WORKSPACE,
          enforce: true,
          policy: {
            case: "reviewRulePolicy",
            value: create(ReviewRulePolicySchema, {
              rules: version === "v1" ? [] : [...STANDARD_RULE_TYPES],
            }),
          },
        },
      });
    } catch {
      // The response middleware reports the failure.
      setSwitching(false);
      return;
    }
    setDevSqlReviewVersion(version);
    globalThis.location.reload();
  };

  return (
    <FormSection
      layout="single-column"
      padding="none"
      title={t("sql-review.version-switch.self")}
      description={t("sql-review.version-switch.description")}
    >
      <SegmentedControl
        value={current}
        options={VERSION_OPTIONS}
        onValueChange={(version) => {
          if (version !== current) setTarget(version);
        }}
        ariaLabel={t("sql-review.version-switch.self")}
        disabled={!hasWorkspacePermissionV2("bb.policies.update") || switching}
      />
      <AlertDialog
        open={target !== undefined}
        onOpenChange={(open) => {
          if (!open && !switching) setTarget(undefined);
        }}
      >
        <AlertDialogContent>
          <AlertDialogTitle>
            {target === "v1"
              ? t("sql-review.version-switch.to-v1.title")
              : t("sql-review.version-switch.to-v2.title")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {target === "v1"
              ? t("sql-review.version-switch.to-v1.description")
              : t("sql-review.version-switch.to-v2.description")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <Button
              appearance="outline"
              disabled={switching}
              onClick={() => setTarget(undefined)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              disabled={switching}
              onClick={() => {
                if (target) void switchVersion(target);
              }}
            >
              {target === "v1"
                ? t("sql-review.version-switch.to-v1.confirm")
                : t("sql-review.version-switch.to-v2.confirm")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </FormSection>
  );
}
