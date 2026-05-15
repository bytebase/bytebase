import { RefreshCcw } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { TaskRunLogViewer } from "@/react/components/task-run-log";
import { Button } from "@/react/components/ui/button";
import {
  Tabs,
  TabsList,
  TabsPanel,
  TabsTrigger,
} from "@/react/components/ui/tabs";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { PlanDetailTaskRunSession } from "./PlanDetailTaskRunSession";

export function PlanDetailTaskRunDetail({
  databaseEngine,
  taskRun,
}: {
  databaseEngine?: Engine;
  taskRun: TaskRun;
}) {
  const { t } = useTranslation();
  const [detailKey, setDetailKey] = useState(0);
  const showSessionTab = databaseEngine === Engine.POSTGRES;
  const showRefreshButton = taskRun.status === TaskRun_Status.RUNNING;

  return (
    <Tabs defaultValue="logs">
      <TabsList className="gap-x-6">
        <TabsTrigger value="logs">{t("issue.task-run.logs")}</TabsTrigger>
        {showSessionTab && (
          <TabsTrigger value="session">
            {t("issue.task-run.session")}
          </TabsTrigger>
        )}
      </TabsList>
      {showRefreshButton && (
        <div className="mt-2 flex justify-end">
          <Button
            size="xs"
            variant="ghost"
            onClick={() => setDetailKey((value) => value + 1)}
          >
            <RefreshCcw className="h-3.5 w-3.5" />
            {t("common.refresh")}
          </Button>
        </div>
      )}
      <TabsPanel value="logs">
        <TaskRunLogViewer
          key={`logs-${taskRun.name}-${taskRun.status}-${detailKey}`}
          taskRunName={taskRun.name}
        />
      </TabsPanel>
      {showSessionTab && (
        <TabsPanel value="session">
          <PlanDetailTaskRunSession
            key={`session-${taskRun.name}-${taskRun.status}-${detailKey}`}
            taskRun={taskRun}
          />
        </TabsPanel>
      )}
    </Tabs>
  );
}
