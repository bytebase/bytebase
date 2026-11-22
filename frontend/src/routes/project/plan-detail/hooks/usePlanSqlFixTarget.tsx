import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useMemo,
  useRef,
} from "react";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/app/router/handles";
import { expandSpecTargets } from "@/lib/plan/check";
import {
  getAgentSqlFixIdentity,
  notifyAgentSqlFixTargetChanged,
  registerAgentSqlFixTarget,
} from "@/modules/agent/logic/sqlFix";
import { useAppStore } from "@/stores/app";
import { usePlanDetailContext } from "../shell/PlanDetailContext";
import { getSpecStatementContent } from "../utils/localSheet";
import {
  createPlanSqlFixTarget,
  type PlanSqlFixEditor,
  type PlanSqlFixEditorSnapshot,
} from "../utils/sqlFixTarget";

type EditorRegistry = Map<string, PlanSqlFixEditor>;
const PlanSqlFixEditorsContext = createContext<EditorRegistry | null>(null);

export function PlanSqlFixTargetProvider({
  children,
  selectedSpecId,
  onSelectedSpecIdChange,
}: {
  children: ReactNode;
  selectedSpecId: string;
  onSelectedSpecIdChange: (specId: string) => void;
}) {
  const page = usePlanDetailContext();
  const dbGroupsByName = useAppStore((state) => state.dbGroupsByName);
  const current = useRef({ page, selectedSpecId, onSelectedSpecIdChange });
  current.current = { page, selectedSpecId, onSelectedSpecIdChange };
  const editors = useMemo<EditorRegistry>(() => new Map(), []);
  useEffect(() => {
    const adapter = createPlanSqlFixTarget({
      getPage: () => current.current.page,
      getIdentity: getAgentSqlFixIdentity,
      getEditor: (specId) => editors.get(specId),
      getDraftSql: (spec) => {
        const content = getSpecStatementContent(spec);
        if (!content) return undefined;
        try {
          return new TextDecoder("utf-8", {
            fatal: true,
            ignoreBOM: true,
          }).decode(content);
        } catch {
          return undefined;
        }
      },
      expandTargets: (spec) =>
        expandSpecTargets(spec, (name) =>
          useAppStore
            .getState()
            .dbGroupsByName[name]?.matchedDatabases.map(
              (database) => database.name
            )
        ),
      fetchSheet: (name) => useAppStore.getState().fetchSheet(name, true),
      reveal: async (specId) => {
        const { page, selectedSpecId, onSelectedSpecIdChange } =
          current.current;
        page.expandPhase("changes");
        if (selectedSpecId === specId) {
          return ![...editors.keys()].some((displayed) => displayed !== specId);
        }
        if (page.isCreating) {
          onSelectedSpecIdChange(specId);
          return true;
        }
        await router.push(
          {
            name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
            params: { projectId: page.projectId, planId: page.planId, specId },
          },
          { preventScrollReset: true }
        );
        const destination = router.currentRoute.value.params;
        return (
          destination.projectId === page.projectId &&
          destination.planId === page.planId &&
          destination.specId === specId
        );
      },
    });
    return registerAgentSqlFixTarget(adapter);
  }, [editors]);
  useEffect(() => {
    notifyAgentSqlFixTargetChanged();
  }, [
    dbGroupsByName,
    page.currentUser,
    page.pageKey,
    page.plan,
    page.project,
    page.readonly,
    page.ready,
  ]);
  return (
    <PlanSqlFixEditorsContext.Provider value={editors}>
      {children}
    </PlanSqlFixEditorsContext.Provider>
  );
}

export function useRegisterPlanSqlFixEditor(
  snapshot: PlanSqlFixEditorSnapshot,
  apply: PlanSqlFixEditor["apply"]
): void {
  const editors = useContext(PlanSqlFixEditorsContext);
  const current = useRef({ snapshot, apply });
  current.current = { snapshot, apply };
  useEffect(() => {
    if (!editors) return;
    const editor: PlanSqlFixEditor = {
      getSnapshot: () => current.current.snapshot,
      apply: (expected, sql) => current.current.apply(expected, sql),
    };
    editors.set(snapshot.specId, editor);
    notifyAgentSqlFixTargetChanged();
    return () => {
      if (editors.get(snapshot.specId) !== editor) return;
      editors.delete(snapshot.specId);
      notifyAgentSqlFixTargetChanged();
    };
  }, [editors, snapshot.specId]);
  const targetSignature = JSON.stringify(snapshot.targets);
  useEffect(() => {
    if (editors) notifyAgentSqlFixTargetChanged();
  }, [
    editors,
    snapshot.currentSql,
    snapshot.pageKey,
    snapshot.plan,
    snapshot.sheet,
    snapshot.status,
    targetSignature,
  ]);
}
