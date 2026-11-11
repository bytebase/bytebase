import { SHA256 } from "crypto-js";
import type { AgentDiagnosticContext } from "@/modules/agent/logic/diagnosticHelp";
import type {
  AgentSqlFixSource,
  AgentSqlFixTargetAdapter,
  AgentSqlFixTargetState,
} from "@/modules/agent/logic/sqlFix";
import type { Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import type { Sheet } from "@/types/proto-es/v1/sheet_service_pb";
import { hasProjectPermissionV2 } from "@/utils/iam/permission";
import { isSheetContentComplete } from "@/utils/v1/sheet";
import type { PlanDetailPageState } from "../shell/hooks/types";

export interface PlanSqlFixEditorSnapshot extends AgentSqlFixTargetState {
  pageKey: string;
  plan: string;
  specId: string;
  sheet: string;
  targets: readonly string[];
}

export interface PlanSqlFixEditor {
  getSnapshot: () => PlanSqlFixEditorSnapshot;
  apply: (expectedSql: string, sql: string) => boolean;
}

interface PlanSqlFixTargetDeps {
  getPage: () => PlanDetailPageState;
  getIdentity: () => string | undefined;
  getEditor: (specId: string) => PlanSqlFixEditor | undefined;
  getDraftSql: (spec: Plan_Spec) => string | undefined;
  expandTargets: (spec: Plan_Spec) => string[];
  fetchSheet: (name: string) => Promise<Sheet | undefined>;
  reveal: (specId: string) => boolean | void | Promise<boolean | void>;
}

const sameTargets = (left: readonly string[], right: readonly string[]) => {
  const sortedRight = [...right].sort((a, b) => a.localeCompare(b));
  return (
    left.length === right.length &&
    [...left]
      .sort((a, b) => a.localeCompare(b))
      .every((target, index) => target === sortedRight[index])
  );
};

const sheetOf = (spec: Plan_Spec) =>
  spec.config.case === "changeDatabaseConfig" && !spec.config.value.release
    ? spec.config.value.sheet
    : undefined;

const identityOwnsPage = (identity: string, page: PlanDetailPageState) => {
  try {
    const owner: unknown = JSON.parse(identity);
    return Array.isArray(owner) && owner[1] === page.currentUser.name;
  } catch {
    return false;
  }
};

const editState = (
  page: PlanDetailPageState
): AgentSqlFixTargetState["status"] => {
  if (!page.ready) return "loading";
  if (page.readonly || page.plan.hasRollout) return "readonly";
  if (
    !page.isCreating &&
    page.currentUser.name !== page.plan.creator &&
    !hasProjectPermissionV2(page.project, "bb.plans.update")
  )
    return "permission-denied";
  return "ready";
};

export function createPlanSqlFixTarget({
  getPage,
  getIdentity,
  getEditor,
  getDraftSql,
  expandTargets,
  fetchSheet,
  reveal,
}: PlanSqlFixTargetDeps): AgentSqlFixTargetAdapter {
  const sourceOnPage = (source: AgentSqlFixSource) => {
    const page = getPage();
    return (
      Boolean(source.identity) &&
      source.identity === getIdentity() &&
      identityOwnsPage(source.identity, page) &&
      source.pageKey === page.pageKey &&
      source.plan === page.plan.name &&
      source.resource.project === page.project.name
    );
  };

  const inspect = (source: AgentSqlFixSource): AgentSqlFixTargetState => {
    if (!sourceOnPage(source)) return { status: "stale" };
    const page = getPage();
    const status = editState(page);
    if (status !== "ready") return { status };
    const spec = page.plan.specs.find(
      (candidate) => candidate.id === source.specId
    );
    if (
      !spec ||
      sheetOf(spec) !== source.sheet ||
      !sameTargets(expandTargets(spec), source.targets)
    )
      return { status: "stale" };
    const editor = getEditor(source.specId);
    if (!editor) return { status: "loading" };
    const current = editor.getSnapshot();
    if (
      current.pageKey !== source.pageKey ||
      current.plan !== source.plan ||
      current.specId !== source.specId ||
      current.sheet !== source.sheet ||
      !sameTargets(current.targets, source.targets)
    )
      return { status: "stale" };
    if (current.status !== "ready") return { status: current.status };
    if (current.currentSql !== source.sql) return { status: "stale" };
    return { status: "ready", currentSql: current.currentSql };
  };

  return {
    capture: async (context: AgentDiagnosticContext) => {
      const page = getPage();
      const identity = getIdentity();
      const hint = context.sqlFixSource;
      if (
        !identity ||
        !identityOwnsPage(identity, page) ||
        !page.ready ||
        !hint ||
        !/^[a-f0-9]{64}$/i.test(hint.sha256) ||
        context.resource.project !== page.project.name ||
        !context.sourceId
      )
        return undefined;
      const hash = hint.sha256.toLowerCase();
      const candidates = page.plan.specs.filter((spec) => {
        const sheet = sheetOf(spec);
        if (!sheet || (hint.specId && spec.id !== hint.specId)) return false;
        if (!hint.specId && sheet !== `${page.project.name}/sheets/${hash}`)
          return false;
        const targets = expandTargets(spec);
        return !hint.target || targets.includes(hint.target);
      });
      if (candidates.length !== 1) return undefined;
      const spec = candidates[0];
      const targets = expandTargets(spec);
      if (
        (context.resource.kind === "plan" &&
          context.resource.name !== page.plan.name) ||
        (context.resource.kind === "database" &&
          !targets.includes(context.resource.name)) ||
        !["plan", "database"].includes(context.resource.kind)
      )
        return undefined;
      let sql: string;
      let sheet: string;
      if (context.sql?.source === "draft") {
        sql = context.sql.text;
        sheet = sheetOf(spec)!;
        if (
          context.sql.sheet !== sheet ||
          context.sql.sha256?.toLowerCase() !== hash ||
          SHA256(sql).toString() !== hash ||
          (getEditor(spec.id)?.getSnapshot().currentSql ??
            getDraftSql(spec)) !== sql
        )
          return undefined;
      } else {
        sheet = `${page.project.name}/sheets/${hash}`;
        const original = await fetchSheet(sheet);
        if (
          !original ||
          original.name !== sheet ||
          !isSheetContentComplete(original)
        )
          return undefined;
        try {
          sql = new TextDecoder("utf-8", {
            fatal: true,
            ignoreBOM: true,
          }).decode(original.content);
        } catch {
          return undefined;
        }
        if (SHA256(sql).toString() !== hash) return undefined;
      }
      const source: AgentSqlFixSource = {
        identity,
        pageKey: page.pageKey,
        resource: { ...context.resource },
        sourceId: context.sourceId,
        plan: page.plan.name,
        specId: spec.id,
        sheet,
        targets: [...targets],
        sha256: hash,
        sql,
      };
      if (!sourceOnPage(source)) return undefined;
      return source;
    },
    reveal: (source) => {
      if (sourceOnPage(source)) return reveal(source.specId);
    },
    inspect,
    apply: (source, sql) => {
      if (inspect(source).status !== "ready") return false;
      return getEditor(source.specId)?.apply(source.sql, sql) ?? false;
    },
  };
}
