import { ConnectError } from "@connectrpc/connect";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { createBehaviorMetric } from "@/app/analytics/behavior";
import { behaviorAnalytics } from "@/app/analytics/provider";
import { SplashPanel } from "@/app/layouts/SplashLayout";
import { router } from "@/app/router";
import {
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DETAIL,
} from "@/app/router/handles";
import { Button } from "@/components/ui/button";
import {
  useCurrentUser,
  useIntroStateByKey,
  useWorkspace,
  useWorkspacePermission,
} from "@/hooks/useAppState";
import {
  CREATE_PROJECT_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
} from "@/lib/productIntro";
import {
  saveGuideTeamSize,
  saveSelectedGuideScenarioId,
} from "@/modules/workspace-setup-guide/selection";
import { saveWorkspaceSetupFinished } from "@/modules/workspace-setup-guide/setup";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import type { ValidatedMessage } from "@/types";
import { extractProjectResourceName } from "@/utils/v1/project";
import {
  clearSetupDraft,
  projectOptional,
  readSetupDraft,
  type SetupDraft,
  type SetupSource,
  saveSetupDraft,
  watchSetupDraft,
} from "./workspace-setup/draft";
import {
  findLostCreate,
  runSetupFinish,
  type SetupNames,
  saveSetupNames,
} from "./workspace-setup/finish";
import { GoalScreen } from "./workspace-setup/GoalScreen";
import { validInviteAddresses } from "./workspace-setup/inviteEmails";
import {
  ProjectScreen,
  type SampleOffer,
} from "./workspace-setup/ProjectScreen";
import {
  guessDisplayName,
  guessWorkspaceTitle,
} from "./workspace-setup/prefill";
import { SetupPanel } from "./workspace-setup/SetupPanel";
import {
  type SetupFinishFailure,
  TeamScreen,
} from "./workspace-setup/TeamScreen";
import { WorkspaceScreen } from "./workspace-setup/WorkspaceScreen";

const WORKSPACE_SETUP_PAGE_ENTERED_KEY = "workspace-setup.page-entered.v1";
const WORKSPACE_SETUP_VERSION = "v2";

// The server's own words for a failure, when it gave any.
const serverReason = (error: unknown) =>
  error instanceof ConnectError ? error.rawMessage : undefined;

export function WorkspaceSetupPage() {
  const currentUser = useCurrentUser();
  // The draft is kept per account, which is known only once the user loads.
  if (!currentUser.email) return null;
  return <WorkspaceSetupWizard key={currentUser.email} />;
}

function WorkspaceSetupWizard() {
  const { t } = useTranslation();
  const currentUser = useCurrentUser();
  const workspace = useWorkspace();
  const workspacePolicy = useAppStore((state) => state.workspacePolicy);
  const isSaaSMode = useAppStore((state) => state.isSaaSMode());
  const sample = useAppStore((state) => state.serverInfo?.sample);
  const canUpdateWorkspace = useWorkspacePermission("bb.workspaces.update");
  const setupPageEntered = useIntroStateByKey(WORKSPACE_SETUP_PAGE_ENTERED_KEY);
  const setupPageEnteredRef = useRef(false);
  const [draft, setDraft] = useState<SetupDraft>(() =>
    readSetupDraft(isSaaSMode)
  );
  // One tab works on setup at a time: the one that saved the draft last. When
  // another tab saves it or ends setup, this one shows a notice in place of
  // its screen. Then, or once setup ends here or the wizard is left, what it
  // had started writes nothing more (D11).
  const [elsewhere, setElsewhere] = useState<"saved" | "ended">();
  const lostHolds = useRef(0);
  useEffect(() => {
    const stopWatching = watchSetupDraft((ended) => {
      lostHolds.current += 1;
      setElsewhere(ended ? "ended" : "saved");
    });
    return () => {
      stopWatching();
      lostHolds.current += 1;
    };
  }, []);
  // For work starting now: true once this tab no longer holds setup.
  const supersededFromNow = () => {
    const at = lostHolds.current;
    return () => lostHolds.current !== at;
  };
  const endSetup = () => {
    lostHolds.current += 1;
    clearSetupDraft();
  };
  // Written through on every answer, so a reload, or a tab closed while Finish
  // runs, loses nothing: least of all that the project was created.
  const draftRef = useRef(draft);
  const update = (patch: Partial<SetupDraft>) => {
    const next = { ...draftRef.current, ...patch };
    draftRef.current = next;
    setDraft(next);
    saveSetupDraft(next);
    setElsewhere(undefined);
  };
  // False only while the ID field shows why. Its checks answer after each
  // change, and Finish has the server's word on an ID that got past them.
  const [projectIdValid, setProjectIdValid] = useState(true);
  const [finishing, setFinishing] = useState(false);
  const [failure, setFailure] = useState<SetupFinishFailure>();
  const [skipping, setSkipping] = useState(false);
  const [skipFailure, setSkipFailure] = useState<{ detail?: string }>();

  const idTaken = t("resource-id.validation.duplicated", {
    resource: t("common.project"),
  });
  const validateProjectId = async (id: string): Promise<ValidatedMessage[]> => {
    const taken: ValidatedMessage[] = [{ type: "error", message: idTaken }];
    const { attemptedProjectId, projectTitle } = draftRef.current;
    if (attemptedProjectId !== id) {
      const existing = await useAppStore
        .getState()
        .fetchProject(`${projectNamePrefix}${id}`, true);
      return existing ? taken : [];
    }
    // A create for this ID got no answer. A project that holds the ID settles
    // it: the draft's own exists from here on, and another's leaves nothing of
    // the draft's behind. No project settles nothing, as the create may still
    // be on its way.
    const superseded = supersededFromNow();
    try {
      const found = await findLostCreate(
        id,
        projectTitle.trim(),
        currentUser.email
      );
      // The answer is about the name and ID it was asked with; one typed
      // while it was out is not that project's, and a draft this tab no longer
      // holds is not its to settle.
      const now = draftRef.current;
      if (
        found.kind === "absent" ||
        superseded() ||
        now.projectId !== id ||
        now.projectTitle !== projectTitle
      ) {
        return [];
      }
      update({
        attemptedProjectId: undefined,
        ...(found.kind === "own" ? { createdProject: found.project } : {}),
      });
      return found.kind === "other" ? taken : [];
    } catch {
      // Still unknown; Finish asks the server again.
      return [];
    }
  };

  // Only the sole member, who just created the workspace, names it; an
  // invited account joins a workspace that already has its name.
  const canRenameWorkspace =
    canUpdateWorkspace &&
    (workspacePolicy?.bindings ?? []).reduce(
      (count, binding) => count + binding.members.length,
      0
    ) === 1;
  const guessedName = guessDisplayName(currentUser.email, currentUser.title);
  const guessedWorkspaceTitle = () => {
    const guess = guessWorkspaceTitle(
      currentUser.email,
      guessedName,
      workspace?.title
    );
    switch (guess.kind) {
      case "title":
        return guess.title;
      case "personal":
        return t("workspace-setup.workspace.default-name", {
          name: guess.firstName,
        });
      case "anonymous":
        return t("workspace-setup.workspace.default-name-anonymous");
    }
  };
  // A sample the server lists under this draft's own project is not one
  // another project used: a reload cut its request short, and the stored
  // choice still decides whether to ask for it again.
  const sampleUsedElsewhere = !!sample?.instances.some(
    ({ instance }) =>
      !draft.createdProject || !instance.startsWith(`${draft.createdProject}/`)
  );
  const sampleOffer: SampleOffer = !sample?.available
    ? "unavailable"
    : sampleUsedElsewhere
      ? "used"
      : "offered";
  // Once its request has returned, the sample is the project's database
  // whatever the server offers now.
  const source: SetupSource = draft.samplePrepared
    ? "sample"
    : sampleOffer === "offered"
      ? draft.source
      : "own";
  const name = draft.name ?? guessedName;
  const workspaceTitle = draft.workspaceTitle ?? guessedWorkspaceTitle();

  useEffect(() => {
    if (
      !currentUser.name ||
      !workspace?.name ||
      setupPageEntered ||
      setupPageEnteredRef.current
    ) {
      return;
    }
    setupPageEnteredRef.current = true;
    try {
      useAppStore.getState().saveIntroStateByKey({
        key: WORKSPACE_SETUP_PAGE_ENTERED_KEY,
        newState: true,
      });
    } catch {
      // Analytics markers must not interrupt workspace setup when storage fails.
    }
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("workspace setup page entered", {
        properties: { setup_version: WORKSPACE_SETUP_VERSION },
      })
    );
  }, [currentUser.name, setupPageEntered, workspace?.name]);

  // A failed attempt is reported for the answers it ran with, so the report
  // goes when its screen is left, and a failure adding the people listed goes
  // when the team answers change. Any other step's failure still stands then.
  const show = (screen: SetupDraft["screen"]) => {
    setFailure(undefined);
    setSkipFailure(undefined);
    update({ screen });
  };
  const answerTeam = (
    patch: Partial<Pick<SetupDraft, "teamSize" | "emails">>
  ) => {
    if (failure?.stage === "invitees") setFailure(undefined);
    update(patch);
  };

  // Takes the stored draft up only when asked: a screen's fields and a failure
  // report hold state that a draft taken in underneath them would not match.
  const continueHere = () => {
    draftRef.current = readSetupDraft(isSaaSMode);
    // Saved at once, so a tab still working on the draft stops there.
    show(draftRef.current.screen);
    setProjectIdValid(true);
  };

  // Emitted on the click, so a leaver can be placed on a screen. No event
  // carries a name, an address or a project name.
  const continueFrom = (screen: 1 | 2 | 3, next: SetupDraft["screen"]) => {
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("workspace setup screen continued", {
        properties: {
          setup_version: WORKSPACE_SETUP_VERSION,
          screen,
          goal: draft.goal ?? "unselected",
          project_named: !!draft.projectTitle.trim(),
          ...(screen === 3 ? { source } : {}),
        },
      })
    );
    show(next);
  };

  const captureSubmitted = (properties: Record<string, unknown>) =>
    behaviorAnalytics.captureMetric(
      createBehaviorMetric("workspace setup submitted", {
        properties: {
          setup_version: WORKSPACE_SETUP_VERSION,
          scenario: draft.goal ?? "unselected",
          ...properties,
        },
      })
    );

  const names: SetupNames = {
    userName: currentUser.name,
    displayName: name.trim(),
    workspaceName: canRenameWorkspace ? workspace?.name : undefined,
    workspaceTitle: workspaceTitle.trim(),
  };

  // The projects page, which offers to create the first one.
  const leaveWithoutProject = () =>
    router.replace({
      name: PROJECT_V1_ROUTE_DASHBOARD,
      query: { [PRODUCT_INTRO_QUERY_KEY]: CREATE_PROJECT_PRODUCT_INTRO },
    });

  // Self-hosted only: setup is offered once there and may be left for later.
  const skip = async () => {
    const superseded = supersededFromNow();
    setSkipping(true);
    setSkipFailure(undefined);
    try {
      await saveSetupNames(names, superseded);
    } catch (error) {
      if (!superseded()) setSkipFailure({ detail: serverReason(error) });
      setSkipping(false);
      return;
    }
    if (superseded()) {
      setSkipping(false);
      return;
    }
    endSetup();
    if (draft.goal) saveSelectedGuideScenarioId(draft.goal);
    captureSubmitted({ result: "skipped", sample_enabled: false });
    leaveWithoutProject();
  };

  const mayLeaveProject = projectOptional(draft, isSaaSMode);
  const withoutProject = mayLeaveProject && !draft.projectTitle.trim();
  // The addresses Finish adds. Self-hosted lists no one (D7), whatever an
  // earlier draft holds.
  const invitees =
    !isSaaSMode || draft.teamSize === "just-me"
      ? []
      : validInviteAddresses(draft.emails, currentUser.email);

  const finish = async ({ withoutSample = false } = {}) => {
    if (!draft.teamSize) return;
    // Saved first, so another tab on the draft stops before this attempt
    // writes.
    update(withoutSample ? { source: "own" } : {});
    const superseded = supersededFromNow();
    // What the attempt learns is saved only while this tab holds setup.
    const save = (patch: Partial<SetupDraft>) => {
      if (!superseded()) update(patch);
    };
    setFinishing(true);
    setFailure(undefined);
    const finishSource: SetupSource = withoutSample ? "own" : source;
    const answers = {
      source: withoutProject ? "none" : finishSource,
      team_size: draft.teamSize,
      invite_count: invitees.length,
      // Kept beside `team_size` for the dashboards built on the earlier
      // wizard's team-or-solo answer.
      collaboration_type: draft.teamSize === "just-me" ? "solo" : "team",
      sample_enabled: !withoutProject && finishSource === "sample",
    };
    const result = await runSetupFinish(
      {
        ...names,
        withoutProject,
        projectTitle: draft.projectTitle.trim(),
        projectId: draft.projectId,
        createdProject: draft.createdProject,
        ownerEmail: currentUser.email,
        createAttempted: draft.attemptedProjectId === draft.projectId,
        // Asked again until the request has returned once: a self-hosted
        // server lists the instance before it is ready, and a repeated
        // request answers with the same one.
        prepareSample:
          !withoutProject && finishSource === "sample" && !draft.samplePrepared,
        invitees,
        projectGrants: draft.projectGrants ?? {},
        superseded,
      },
      {
        projectCreateSent: () => save({ attemptedProjectId: draft.projectId }),
        projectCreated: (createdProject) => save({ createdProject }),
        samplePrepared: () => save({ samplePrepared: true }),
        projectGrants: (projectGrants) => save({ projectGrants }),
      }
    );
    if ("superseded" in result) {
      // The notice that took the screen's place shows from here.
      setFinishing(false);
      return;
    }
    if (!result.ok) {
      captureSubmitted({
        ...answers,
        result: "failed",
        failure_stage: result.stage,
      });
      if (result.refusal) {
        // The server settled that create: nothing of this draft's holds the ID.
        update({
          attemptedProjectId: undefined,
          refusedProjectId: { id: draft.projectId, ...result.refusal },
        });
        show("project");
      } else {
        setFailure({
          stage: result.stage,
          detail: serverReason(result.error),
        });
      }
      setFinishing(false);
      return;
    }
    captureSubmitted({ ...answers, result: "finished" });
    endSetup();
    if (draft.goal) saveSelectedGuideScenarioId(draft.goal);
    saveGuideTeamSize(draft.teamSize);
    if (isSaaSMode) {
      saveWorkspaceSetupFinished(
        workspace?.name ?? currentUser.workspace,
        true
      );
    }
    if (!result.project) {
      leaveWithoutProject();
      return;
    }
    try {
      useAppStore.getState().setRecentProject(result.project);
    } catch {
      // Everything exists on the server; a browser that cannot store the
      // recent project must still leave the wizard.
    }
    router.replace({
      name: PROJECT_V1_ROUTE_DETAIL,
      params: { projectId: extractProjectResourceName(result.project) },
    });
  };

  const renderScreen = () => {
    switch (draft.screen) {
      case "workspace":
        return (
          <WorkspaceScreen
            name={name}
            workspaceTitle={workspaceTitle}
            showWorkspaceTitle={canRenameWorkspace}
            onNameChange={(value) => update({ name: value })}
            onWorkspaceTitleChange={(value) =>
              update({ workspaceTitle: value })
            }
            onContinue={() => continueFrom(1, "goal")}
          />
        );
      case "goal":
        return (
          <GoalScreen
            goal={draft.goal}
            onGoalChange={(goal) => update({ goal })}
            onBack={() => show("workspace")}
            onContinue={() => continueFrom(2, "project")}
          />
        );
      case "team":
        return (
          <TeamScreen
            teamSize={draft.teamSize}
            addresses={draft.emails}
            ownerEmail={currentUser.email}
            projectTitle={draft.projectTitle.trim()}
            source={source}
            isSaaS={isSaaSMode}
            onTeamSizeChange={(teamSize) => answerTeam({ teamSize })}
            onAddressesChange={(emails) => answerTeam({ emails })}
            finishing={finishing}
            failure={failure}
            onBack={() => show("project")}
            onFinish={() => void finish()}
            onFinishWithoutSample={() => void finish({ withoutSample: true })}
          />
        );
      case "project":
        return (
          <ProjectScreen
            projectTitle={draft.projectTitle}
            projectId={draft.projectId}
            projectIdValid={projectIdValid}
            projectIdRefusal={
              draft.refusedProjectId?.id === draft.projectId
                ? (draft.refusedProjectId.reason ?? idTaken)
                : undefined
            }
            projectExists={!!draft.createdProject}
            source={source}
            sample={sampleOffer}
            samplePrepared={!!draft.samplePrepared}
            isSaaS={isSaaSMode}
            projectOptional={mayLeaveProject}
            skipping={skipping}
            skipFailure={skipFailure}
            onSkip={mayLeaveProject ? () => void skip() : undefined}
            validateProjectId={validateProjectId}
            onProjectTitleChange={(projectTitle) => {
              if (projectTitle.trim()) {
                update({ projectTitle });
              } else {
                // An unnamed project has no ID, and so no verdict on one; the
                // next name derives a new ID.
                update({ projectTitle, projectId: "" });
                setProjectIdValid(true);
              }
            }}
            onProjectIdChange={(projectId) => {
              if (draftRef.current.projectId !== projectId) {
                update({ projectId });
              }
            }}
            onProjectIdValidChange={setProjectIdValid}
            onSourceChange={(source) => update({ source })}
            onBack={() => show("goal")}
            onContinue={() => continueFrom(3, "team")}
          />
        );
    }
  };

  return (
    <>
      <SplashPanel>
        <SetupPanel
          screen={draft.screen}
          goal={draft.goal}
          workspaceTitle={
            canRenameWorkspace ? workspaceTitle : (workspace?.title ?? "")
          }
          memberName={name.trim() || currentUser.title || currentUser.email}
          memberEmail={currentUser.email}
          projectTitle={draft.projectTitle.trim()}
          source={source}
          noProject={withoutProject}
          teamSize={draft.teamSize}
          invitees={invitees}
        />
      </SplashPanel>
      {elsewhere && !finishing ? (
        <div className="mx-auto flex w-full max-w-lg flex-col items-start gap-y-4">
          <h1 className="text-xl font-semibold text-main">
            {elsewhere === "ended"
              ? t("workspace-setup.elsewhere.ended")
              : t("workspace-setup.elsewhere.saved")}
          </h1>
          {elsewhere === "ended" ? (
            <Button
              onClick={() =>
                router.replace({ name: PROJECT_V1_ROUTE_DASHBOARD })
              }
            >
              {t("workspace-setup.elsewhere.workspace")}
            </Button>
          ) : (
            <Button onClick={continueHere}>
              {t("workspace-setup.elsewhere.continue")}
            </Button>
          )}
        </div>
      ) : (
        renderScreen()
      )}
    </>
  );
}
