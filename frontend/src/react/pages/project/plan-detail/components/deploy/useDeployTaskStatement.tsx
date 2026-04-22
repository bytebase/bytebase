import { useEffect, useMemo, useState } from "react";
import { useSheetV1Store } from "@/store";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import { getStatementSize } from "@/utils/sheet";
import { sheetNameOfTaskV1 } from "@/utils/v1/issue/rollout";
import { getSheetStatement } from "@/utils/v1/sheet";

export const useDeployTaskStatement = ({
  enabled,
  task,
}: {
  enabled: boolean;
  task: Task;
}) => {
  const sheetStore = useSheetV1Store();
  const sheetName = useMemo(() => sheetNameOfTaskV1(task), [task]);
  const [statement, setStatement] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [isTruncated, setIsTruncated] = useState(false);

  useEffect(() => {
    if (!enabled || !sheetName) {
      setStatement("");
      setIsTruncated(false);
      setIsLoading(false);
      return;
    }

    let canceled = false;
    const load = async () => {
      setIsLoading(true);
      try {
        let sheet = sheetStore.getSheetByName(sheetName);
        if (!sheet) {
          sheet = await sheetStore.getOrFetchSheetByName(sheetName);
        }
        if (!sheet) {
          return;
        }
        if (canceled) return;
        const nextStatement = getSheetStatement(sheet);
        setStatement(nextStatement);
        setIsTruncated(getStatementSize(nextStatement) < sheet.contentSize);
      } finally {
        if (!canceled) {
          setIsLoading(false);
        }
      }
    };
    void load();
    return () => {
      canceled = true;
    };
  }, [enabled, sheetName, sheetStore]);

  return { isLoading, isTruncated, statement };
};
