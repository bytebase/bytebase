import { useLayoutEffect, useRef } from "react";
import type { IStandaloneCodeEditor } from "@/components/monaco/types";
import { registerAgentDomOwner } from "@/modules/agent/dom/interactionRegistry";

type StatementOwner = {
  actor: string;
  workspace: string;
  pageKey: string;
  plan: string;
  project: string;
  source: object;
  reader?: IStandaloneCodeEditor;
  mode: "read" | "local" | "manual";
  draft: string;
  begin: () => void;
  input: (sql: string) => void;
  cancel: () => void;
};

function canKeepFocus(plan: string, input?: Element | null) {
  const active = document.activeElement;
  return (
    !active ||
    active === document.body ||
    active === input ||
    (active instanceof HTMLInputElement &&
      active.dataset.planTitleInput === plan) ||
    (active.closest("[data-bb-layer-family=agent]") !== null &&
      !active.closest("[data-bb-layer-family=critical]"))
  );
}

export function usePlanStatementDomOwner(owner: StatementOwner) {
  const edit = useRef<HTMLButtonElement>(null);
  const input = useRef<HTMLTextAreaElement>(null);
  const cancel = useRef<HTMLButtonElement>(null);
  const readerGround = useRef<HTMLDivElement>(null);
  const readerReady = useRef<() => boolean>(() => false);
  const live = useRef(owner);
  const epoch = useRef(0);
  const { reader } = owner;
  useLayoutEffect(() => {
    const node = reader?.getDomNode();
    const model = reader?.getModel();
    if (!reader || !node || !model || !readerGround.current?.contains(node))
      return;
    // Monaco can recreate its editor inside the same mounted reader. Losing
    // this ready occurrence is terminal, even if its cached model is reused.
    let retired = false;
    const retire = () => {
      if (!retired) epoch.current++;
      retired = true;
    };
    readerReady.current = () => {
      if (retired) return false;
      if (
        !node.isConnected ||
        !readerGround.current?.contains(node) ||
        reader.getDomNode() !== node ||
        reader.getModel() !== model ||
        model.isDisposed()
      )
        retire();
      return !retired;
    };
    const disposed = reader.onDidDispose(retire);
    const changedModel = reader.onDidChangeModel(retire);
    return () => {
      retire();
      disposed.dispose();
      changedModel.dispose();
      readerReady.current = () => false;
    };
  }, [reader]);
  useLayoutEffect(() => {
    const old = live.current;
    if (
      old.source !== owner.source ||
      old.reader !== owner.reader ||
      old.mode !== owner.mode ||
      old.draft !== owner.draft
    )
      epoch.current++;
    live.current = owner;
  });
  const { actor, workspace, plan, project, pageKey } = owner;
  useLayoutEffect(() => {
    if (!actor || !workspace || !plan || !project) return;
    return registerAgentDomOwner({
      ownerId: "plan-statement",
      actor,
      workspace,
      getSnapshot: () => ({
        pageKey,
        resourceName: plan,
        projectName: project,
        revision: String(epoch.current),
      }),
      getBinding(el, verb) {
        const state = live.current;
        const action =
          verb === "click" && el === edit.current && state.mode === "read"
            ? "begin"
            : verb === "input" && el === input.current && state.mode === "local"
              ? "input"
              : verb === "click" &&
                  el === cancel.current &&
                  state.mode === "local"
                ? "cancel"
                : undefined;
        if (!action) return;
        return {
          id: action,
          verb,
          effect: "local-draft",
          eligible: () =>
            readerReady.current() &&
            canKeepFocus(plan, action === "input" ? input.current : null),
          run(value) {
            if (action === "input") {
              if (typeof value !== "string")
                return { success: false, message: "Missing SQL draft." };
              state.input(value);
            } else {
              state[action]();
            }
            return {
              success: true,
              message:
                "Updated the local Plan statement draft. Nothing was saved.",
            };
          },
        };
      },
    });
  }, [actor, workspace, plan, project, pageKey]);
  return { edit, input, cancel, readerGround };
}
