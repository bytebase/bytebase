import { useMemo } from "react";
import { useSheetStatement } from "@/react/hooks/useSheetStatement";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { sheetNameOfTaskV1 } from "@/utils/v1/issue/rollout";

// Rollout tasks always reference committed (remote) sheets, so this is just the
// shared sheet-statement loader keyed on the task's sheet — seeded from cache to
// avoid the first-paint "No data" flash when expanding a task or switching
// stages (BYT-9763).
export const useDeployTaskStatement = ({
  enabled,
  task,
}: {
  enabled: boolean;
  task: Task;
}) => {
  const sheetName = useMemo(() => sheetNameOfTaskV1(task), [task]);
  return useSheetStatement({ enabled, sheetName });
};
