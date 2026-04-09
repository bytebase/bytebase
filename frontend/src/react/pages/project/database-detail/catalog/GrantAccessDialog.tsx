import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rewriteResourceDatabase } from "@/components/SensitiveData/exemptionDataUtils";
import type { SensitiveColumn } from "@/components/SensitiveData/types";
import {
  convertSensitiveColumnToDatabaseResource,
  getExpressionsForDatabaseResource,
} from "@/components/SensitiveData/utils";
import { AccountMultiSelect } from "@/react/components/AccountMultiSelect";
import { DatabaseResourceSelector } from "@/react/components/DatabaseResourceSelector";
import { Button } from "@/react/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/react/components/ui/dialog";
import { ExpirationPicker } from "@/react/components/ui/expiration-picker";
import { Input } from "@/react/components/ui/input";
import { pushNotification, usePolicyV1Store } from "@/store";
import type { DatabaseResource } from "@/types";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import {
  MaskingExemptionPolicy_ExemptionSchema,
  MaskingExemptionPolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";

export interface GrantAccessDialogProps {
  columnList: SensitiveColumn[];
  open: boolean;
  projectName: string;
  onDismiss: () => void;
}

export function GrantAccessDialog({
  columnList,
  open,
  projectName,
  onDismiss,
}: GrantAccessDialogProps) {
  const { t } = useTranslation();
  const policyStore = usePolicyV1Store();

  const [databaseResources, setDatabaseResources] = useState<
    DatabaseResource[]
  >(() => columnList.map(convertSensitiveColumnToDatabaseResource));
  const [description, setDescription] = useState("");
  const [expirationTimestamp, setExpirationTimestamp] = useState<
    string | undefined
  >();
  const [memberList, setMemberList] = useState<string[]>([]);
  const [processing, setProcessing] = useState(false);

  const minDatetime = useMemo(
    () => dayjs().startOf("day").format("YYYY-MM-DDTHH:mm"),
    []
  );
  const submitDisabled =
    memberList.length === 0 || databaseResources.length === 0;

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      onDismiss();
    }
  };

  const handleSubmit = async () => {
    if (submitDisabled || processing) {
      return;
    }
    setProcessing(true);

    try {
      const extraExpressions: string[] = [];
      if (expirationTimestamp) {
        extraExpressions.push(
          `request.time < timestamp("${new Date(expirationTimestamp).toISOString()}")`
        );
      }

      const resourceExpressions = databaseResources
        .map(getExpressionsForDatabaseResource)
        .map((parts) => parts.filter((part) => part).join(" && "));
      const nonEmptyResourceExpressions = resourceExpressions.filter(
        (expression) => expression
      );

      let resourceCondition = "";
      if (nonEmptyResourceExpressions.length === 1) {
        resourceCondition = nonEmptyResourceExpressions[0];
      } else if (nonEmptyResourceExpressions.length > 1) {
        resourceCondition = nonEmptyResourceExpressions
          .map((expression) => `(${expression})`)
          .join(" || ");
      }

      const expression = rewriteResourceDatabase(
        [resourceCondition, ...extraExpressions]
          .filter((part) => part)
          .join(" && ")
      );

      const policy = await policyStore.getOrFetchPolicyByParentAndType({
        parentPath: projectName,
        policyType: PolicyType.MASKING_EXEMPTION,
      });
      const existed =
        policy?.policy?.case === "maskingExemptionPolicy"
          ? policy.policy.value.exemptions
          : [];

      await policyStore.upsertPolicy({
        parentPath: projectName,
        policy: {
          name: policy?.name,
          type: PolicyType.MASKING_EXEMPTION,
          resourceType: PolicyResourceType.PROJECT,
          policy: {
            case: "maskingExemptionPolicy",
            value: create(MaskingExemptionPolicySchema, {
              exemptions: [
                ...existed,
                create(MaskingExemptionPolicy_ExemptionSchema, {
                  members: memberList,
                  condition: create(ExprSchema, {
                    description,
                    expression,
                  }),
                }),
              ],
            }),
          },
        },
      });

      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.created"),
      });
      onDismiss();
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `${error}`,
      });
    } finally {
      setProcessing(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="max-w-4xl p-6">
        <DialogTitle>{t("settings.sensitive-data.grant-access")}</DialogTitle>
        <div className="mt-4 flex flex-col gap-y-6">
          <div className="flex flex-col gap-y-2">
            <span className="text-sm font-medium text-main">
              {t("common.resources")}
            </span>
            <DatabaseResourceSelector
              projectName={projectName}
              value={databaseResources}
              onChange={setDatabaseResources}
            />
          </div>

          <div className="flex flex-col gap-y-2">
            <span className="text-sm font-medium text-main">
              {t("common.reason")}
            </span>
            <Input
              value={description}
              placeholder={t("common.description")}
              onChange={(event) => setDescription(event.target.value)}
            />
          </div>

          <div className="flex flex-col gap-y-2">
            <span className="text-sm font-medium text-main">
              {t("common.expiration")}
            </span>
            <ExpirationPicker
              value={expirationTimestamp}
              minDate={minDatetime}
              onChange={setExpirationTimestamp}
            />
            {!expirationTimestamp && (
              <span className="text-sm text-control-light">
                {t("settings.sensitive-data.never-expires")}
              </span>
            )}
          </div>

          <div className="flex flex-col gap-y-2">
            <span className="text-sm font-medium text-main">
              {t("common.members")}
            </span>
            <AccountMultiSelect value={memberList} onChange={setMemberList} />
          </div>
        </div>

        <div className="mt-6 flex justify-end gap-x-2">
          <Button variant="outline" onClick={onDismiss}>
            {t("common.cancel")}
          </Button>
          <Button
            disabled={submitDisabled || processing}
            onClick={() => void handleSubmit()}
          >
            {t("common.confirm")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
