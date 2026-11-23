import { create } from "@bufbuild/protobuf";
import { ArrowLeftRight } from "lucide-react";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip } from "@/components/ui/tooltip";
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

export function SQLReviewVersionBadge() {
  const { t } = useTranslation();
  const current = devSqlReviewVersion();

  return (
    <Tooltip
      content={t("sql-review.version-switch.description")}
      render={
        <span
          tabIndex={0}
          className="inline-flex shrink-0 rounded-xs focus:outline-hidden focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2"
        />
      }
    >
      <Badge className="whitespace-nowrap">
        {current === "v1"
          ? t("sql-review.version-switch.v1")
          : t("sql-review.version-switch.v2")}
      </Badge>
    </Tooltip>
  );
}

// Dev builds only. The server runs V1's SQL review for a project whose
// standard rules are all off, so switching writes the workspace standard rules
// as well as this browser's choice of pages. The page then reloads, since the
// SQL review gate is read outside React state.
export function SQLReviewVersionSwitch() {
  const { t } = useTranslation();
  const workspace = useWorkspaceResourceName();
  const current = devSqlReviewVersion();
  const next: SqlReviewVersion = current === "v1" ? "v2" : "v1";
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
    <>
      <Button
        appearance="outline"
        className="h-auto whitespace-normal py-2 sm:ml-auto"
        onClick={() => setTarget(next)}
        disabled={!hasWorkspacePermissionV2("bb.policies.update") || switching}
      >
        <ArrowLeftRight size={16} className="shrink-0" aria-hidden="true" />
        {next === "v1"
          ? t("sql-review.version-switch.to-v1.confirm")
          : t("sql-review.version-switch.to-v2.confirm")}
      </Button>
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
    </>
  );
}
