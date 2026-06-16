import { create } from "@bufbuild/protobuf";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { rolloutServiceClientConnect } from "@/connect";
import { Button } from "@/react/components/ui/button";
import { router } from "@/react/router";
import { buildPlanDeployRouteFromRolloutName } from "@/react/router/routeHelpers";
import { pushNotification } from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { isReleaseBackedPlan } from "../utils/spec";

export function PlanDetailDeployFuture() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const [creatingRollout, setCreatingRollout] = useState(false);

  const isGitOpsPlan = useMemo(
    () => isReleaseBackedPlan(page.plan.specs),
    [page.plan.specs]
  );
  const canCreateRollout = Boolean(
    isGitOpsPlan &&
      !page.plan.hasRollout &&
      page.plan.state === State.ACTIVE &&
      page.projectCanCreateRollout
  );

  const createRollout = async () => {
    if (creatingRollout) return;
    try {
      setCreatingRollout(true);
      const createdRollout = await rolloutServiceClientConnect.createRollout(
        create(CreateRolloutRequestSchema, { parent: page.plan.name })
      );
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.created"),
      });
      await page.refreshState();
      void router.push(
        buildPlanDeployRouteFromRolloutName(createdRollout.name)
      );
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: t("common.failed"),
        description: String(error),
      });
    } finally {
      setCreatingRollout(false);
    }
  };

  return (
    <div className="mt-1.5">
      <p className="text-sm text-control-placeholder">
        {isGitOpsPlan
          ? t("plan.phase.deploy-description-gitops")
          : t("plan.phase.deploy-description")}
      </p>
      {canCreateRollout && (
        <div className="mt-3">
          <Button
            disabled={creatingRollout}
            onClick={() => void createRollout()}
            size="sm"
            variant="outline"
          >
            {t("plan.phase.create-rollout-action")}
          </Button>
        </div>
      )}
    </div>
  );
}
