import { toJson } from "@bufbuild/protobuf";
import { useCallback, useLayoutEffect, useRef } from "react";
import { router } from "@/app/router";
import type {
  LabelEditorControl,
  LabelListEditorItem,
} from "@/components/LabelListEditor";
import {
  type AgentDomBinding,
  type AgentDomContext,
  type AgentDomMutationPreview,
  type AgentDomOwner,
  manualResult,
  registerAgentDomOwner,
} from "@/modules/agent/dom/interactionRegistry";
import { isAgentExecutionContextCurrent } from "@/modules/agent/logic/executionContext";
import { useAppStore } from "@/stores/app";
import {
  type Database,
  type UpdateDatabaseRequest,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  convertLabelsToKVList,
  getDatabaseProject,
  hasProjectPermissionV2,
} from "@/utils";
import {
  buildDatabaseEnvironmentUpdate,
  buildDatabaseLabelsUpdate,
  compareKeys,
  environmentFingerprint,
  labelMap,
  labelsChanged,
  labelsFingerprint,
  validRows,
} from "./databaseSettingsActions";
import {
  currentDatabaseContext,
  currentEnvironmentExists,
  readAuthorizedDatabase,
  readDatabase,
  sameDatabaseScope,
} from "./databaseSettingsPermission";

type Field = "labels" | "environment";
interface Panel {
  database: Database;
  rows: LabelListEditorItem[];
  setRows: (rows: LabelListEditorItem[]) => void;
  allowed: boolean;
  actor: string;
  workspace: string;
}
type Control = LabelEditorControl | { kind: "save" | "revert" | "environment" };
function controlId(control: Control) {
  return "row" in control ? `${control.kind}:${control.row}` : control.kind;
}
const dbFingerprint = (field: Field, db: Database) =>
  JSON.stringify([
    db.name,
    db.project,
    db.state,
    field === "labels"
      ? labelsFingerprint(db.labels)
      : environmentFingerprint(db),
  ]);
function allowedSelection(database: Database, value: string) {
  if (!database.instanceResource) return false;
  if (value === "")
    return (
      !!database.effectiveEnvironment && !database.instanceResource.environment
    );
  return useAppStore
    .getState()
    .environmentList.some((env) => `environments/${env.id}` === value);
}
async function prepareUpdate(
  request: UpdateDatabaseRequest,
  preview: AgentDomMutationPreview
) {
  const { prepareApiCall } = await import(
    "@/modules/agent/logic/tools/callApi"
  );
  const prepared = await prepareApiCall({
    operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
    body: toJson(UpdateDatabaseRequestSchema, request) as Record<
      string,
      unknown
    >,
  });
  if (prepared.kind !== "prepared") return { result: manualResult() };
  return { call: prepared.call, preview };
}
function nextLabelRows(
  control: Control,
  rows: LabelListEditorItem[],
  database: Database,
  value?: string
) {
  if (control.kind === "revert") return convertLabelsToKVList(database.labels);
  if (control.kind === "add") return [...rows, { key: "", value: "" }];
  if (!("row" in control) || !rows[control.row]) return;
  if (control.kind === "remove")
    return rows.filter((_, i) => i !== control.row);
  if (typeof value === "string")
    return rows.map((row, i) =>
      i === control.row ? { ...row, [control.kind]: value } : row
    );
}

function preservesLabelFocus(
  control: Control,
  rows: LabelListEditorItem[],
  next: LabelListEditorItem[],
  database: Database,
  focused: Control | undefined
) {
  if (!focused) return true;
  const dirty = labelsChanged(database.labels, next);
  switch (focused.kind) {
    case "save":
      return dirty && validRows(next);
    case "revert":
      return dirty;
    case "add":
      return !dirty || validRows(next);
    case "environment":
      return true;
  }
  // Index keys recycle every subsequent row on remove.
  if (control.kind === "remove") return focused.row < control.row;
  if (control.kind !== "revert") return true;
  return (
    next[focused.row]?.key === rows[focused.row].key &&
    next[focused.row]?.value === rows[focused.row].value
  );
}

function persistBinding(
  field: Field,
  database: Database,
  rows: LabelListEditorItem[],
  isCurrent: () => boolean
): AgentDomBinding {
  const guarded = (context: AgentDomContext): AgentDomContext => ({
    ...context,
    isCurrent: () => context.isCurrent() && isCurrent(),
  });
  async function observe(value: string | undefined, context: AgentDomContext) {
    if (
      field === "environment" &&
      (value === undefined || !allowedSelection(database, value))
    )
      return;
    const fresh = await readAuthorizedDatabase(database, context);
    if (
      !fresh ||
      !currentDatabaseContext(context) ||
      dbFingerprint(field, database) !== dbFingerprint(field, fresh)
    )
      return;
    if (
      field === "environment" &&
      (value === undefined ||
        !allowedSelection(fresh, value) ||
        (value && !(await currentEnvironmentExists(value, context))))
    )
      return;
    if (currentDatabaseContext(context)) return fresh;
  }
  return {
    id: field === "environment" ? "environment" : "save",
    verb: field === "environment" ? "select" : "click",
    effect: "persist",
    eligible: isCurrent,
    async prepare(value, stage) {
      const fresh = await observe(value, guarded(stage));
      if (!fresh) return { result: manualResult() };
      if (field === "environment") {
        if (value === undefined) return { result: manualResult() };
        if (value === (fresh.effectiveEnvironment ?? ""))
          return {
            result: {
              success: true,
              message: "The effective environment is unchanged.",
            },
          };
        return prepareUpdate(
          buildDatabaseEnvironmentUpdate(database.name, value),
          {
            kind: "database-environment",
            changes: [
              {
                field: "environment",
                before: fresh.effectiveEnvironment || null,
                after: value || null,
              },
            ],
          }
        );
      }
      const labels = labelMap(rows);
      const keys = [
        ...new Set([...Object.keys(database.labels), ...Object.keys(labels)]),
      ].sort(compareKeys);
      const changes = keys.flatMap((key) => {
        const before = Object.hasOwn(database.labels, key)
            ? database.labels[key]
            : null,
          after = Object.hasOwn(labels, key) ? labels[key] : null;
        return before === after ? [] : [{ field: key, before, after }];
      });
      return prepareUpdate(buildDatabaseLabelsUpdate(database.name, labels), {
        kind: "database-labels",
        changes,
      });
    },
    async validateFresh(context, call) {
      const value =
        field === "environment"
          ? JSON.parse(call.body).database.environment
          : undefined;
      return !!(await observe(value, guarded(context)));
    },
  };
}

export function useDatabaseSettingsDom(panel: Panel) {
  const {
    actor,
    workspace,
    database: { name, project },
  } = panel;
  const live = useRef(panel);
  const versions = useRef({ labels: 1, environment: 1 });
  const pendingNative = useRef({ labels: 0, environment: 0 });
  const pendingDraft = useRef(false);
  const controls = useRef(new Map<Element, LabelEditorControl>());
  const nodes = useRef(new Map<string, Element>());
  const save = useRef<HTMLButtonElement>(null),
    revert = useRef<HTMLButtonElement>(null),
    environment = useRef<HTMLDivElement>(null);
  const controlRef = useCallback(
    (control: LabelEditorControl, el: HTMLElement | null) => {
      const id = controlId(control);
      const old = nodes.current.get(id);
      if (old) controls.current.delete(old);
      if (el) {
        nodes.current.set(id, el);
        controls.current.set(el, control);
      } else nodes.current.delete(id);
    },
    []
  );
  useLayoutEffect(() => {
    for (const field of ["labels", "environment"] as const) {
      if (
        dbFingerprint(field, live.current.database) !==
          dbFingerprint(field, panel.database) ||
        live.current.allowed !== panel.allowed
      )
        versions.current[field]++;
    }
    live.current = panel;
    pendingDraft.current = false;
  });
  const edit = useCallback((rows: LabelListEditorItem[]) => {
    versions.current.labels++;
    pendingDraft.current = true;
    live.current.setRows(rows);
  }, []);
  const startNative = (field: Field) => {
    versions.current[field]++;
    pendingNative.current[field]++;
    return () => {
      versions.current[field]++;
      pendingNative.current[field]--;
    };
  };
  useLayoutEffect(() => {
    if (!actor || !workspace) return;
    let mounted = true;
    let cacheGeneration = 0;
    const owner = (field: Field): AgentDomOwner => {
      function eligible() {
        const { database, allowed } = live.current;
        const cached = useAppStore.getState().databasesByName[name];
        return (
          mounted &&
          !pendingNative.current[field] &&
          !(field === "labels" && pendingDraft.current) &&
          isAgentExecutionContextCurrent({ actor, workspace }) &&
          !!cached &&
          sameDatabaseScope(database, cached) &&
          dbFingerprint(field, cached) === dbFingerprint(field, database) &&
          allowed &&
          hasProjectPermissionV2(
            getDatabaseProject(database),
            "bb.databases.update"
          )
        );
      }
      function getControl(el: Element): Control | undefined {
        if (field === "environment")
          return el === environment.current
            ? { kind: "environment" }
            : undefined;
        if (el === save.current) return { kind: "save" };
        if (el === revert.current) return { kind: "revert" };
        return controls.current.get(el);
      }
      return {
        ownerId: `database-settings:${field}`,
        actor,
        workspace,
        getSnapshot: () => ({
          resourceName: name,
          projectName: project,
          pageKey: router.currentRoute.value.fullPath,
          revision: String(versions.current[field]),
        }),
        getBinding(el, verb): AgentDomBinding | undefined {
          const control = getControl(el);
          if (!control) return;
          const { kind } = control;
          if (
            verb !==
            (kind === "environment"
              ? "select"
              : kind === "key" || kind === "value"
                ? "input"
                : "click")
          )
            return;
          const id = controlId(control);
          const { database, rows } = live.current;
          const dirty = labelsChanged(database.labels, rows);
          const valid = validRows(rows);
          if (kind !== "save" && kind !== "environment") {
            return {
              id,
              verb,
              effect: "local-draft",
              eligible: () =>
                eligible() &&
                (kind !== "revert" || dirty) &&
                (kind !== "add" || !dirty || valid),
              run(value) {
                if (!eligible()) return manualResult();
                const next = nextLabelRows(control, rows, database, value);
                const active = document.activeElement;
                if (
                  !next ||
                  !preservesLabelFocus(
                    control,
                    rows,
                    next,
                    database,
                    active ? getControl(active) : undefined
                  )
                )
                  return manualResult();
                edit(next);
                return {
                  success: true,
                  message: "Updated the local label draft.",
                };
              },
            };
          }
          const revision = versions.current[field];
          return persistBinding(
            field,
            database,
            rows,
            () =>
              eligible() &&
              versions.current[field] === revision &&
              (field !== "labels" || (dirty && valid))
          );
        },
        async refresh(context, committed) {
          const cached = useAppStore.getState().databasesByName[name];
          const generation = cacheGeneration;
          if (!cached || !sameDatabaseScope(live.current.database, cached))
            return;
          const fresh = await readDatabase(live.current.database, context);
          if (!currentDatabaseContext(context)) return;
          if (
            cacheGeneration !== generation ||
            useAppStore.getState().databasesByName[name] !== cached
          )
            return;
          const committedDraft =
            field === "labels" &&
            String(versions.current.labels) === committed.revision;
          if (committedDraft) edit(convertLabelsToKVList(fresh.labels));
          useAppStore.setState((state) => ({
            databasesByName: { ...state.databasesByName, [name]: fresh },
          }));
        },
      };
    };
    const disposeLabels = registerAgentDomOwner(owner("labels")),
      disposeEnv = registerAgentDomOwner(owner("environment"));
    // React may batch A -> B -> A cache changes into no render at all.
    const unsubscribe = useAppStore.subscribe((next, prev) => {
      if (
        next.environmentList !== prev.environmentList &&
        JSON.stringify(
          next.environmentList.map((e) => e.id).sort(compareKeys)
        ) !==
          JSON.stringify(
            prev.environmentList.map((e) => e.id).sort(compareKeys)
          )
      ) {
        versions.current.environment++;
      }
      const before = prev.databasesByName[name],
        after = next.databasesByName[name];
      if (before === after) return;
      cacheGeneration++;
      for (const field of ["labels", "environment"] as const) {
        if (
          !before ||
          !after ||
          dbFingerprint(field, before) !== dbFingerprint(field, after)
        )
          versions.current[field]++;
      }
    });
    return () => {
      mounted = false;
      unsubscribe();
      disposeEnv();
      disposeLabels();
    };
  }, [actor, workspace, name, project, edit]);
  return { controlRef, save, revert, environment, edit, startNative };
}
