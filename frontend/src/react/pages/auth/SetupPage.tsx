import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { Code } from "@connectrpc/connect";
import { Loader2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { AuthFooter } from "@/react/components/auth/AuthFooter";
import { ComponentPermissionGuard } from "@/react/components/ComponentPermissionGuard";
import {
  ResourceIdField,
  type ResourceIdFieldRef,
} from "@/react/components/ResourceIdField";
import { Button } from "@/react/components/ui/button";
import { Input } from "@/react/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/react/components/ui/radio-group";
import { useVueState } from "@/react/hooks/useVueState";
import { router } from "@/router";
import { SQL_EDITOR_HOME_MODULE } from "@/router/sqlEditor";
import {
  useActuatorV1Store,
  useAppFeature,
  useProjectV1Store,
  useRoleStore,
  useSettingV1Store,
  useWorkspaceV1Store,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { DatabaseChangeMode } from "@/types/proto-es/v1/setting_service_pb";
import { unknownProject } from "@/types/v1/project";
import { extractGrpcErrorMessage, getErrorCode } from "@/utils/connect";

type Purpose = "edit-schema" | "query-data";
type Workflow = "simple" | "team";
type DataChoice = "self-setup" | "builtin-sample";

const SETUP_PERMISSIONS = [
  "bb.settings.get",
  "bb.settings.setWorkspaceProfile",
  "bb.projects.create",
  "bb.roles.list",
  "bb.workspaces.getIamPolicy",
] as const;

const homePath = (mode?: DatabaseChangeMode) => {
  if (mode === DatabaseChangeMode.EDITOR) {
    return { name: SQL_EDITOR_HOME_MODULE };
  }
  return "/";
};

export function SetupPage() {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    (async () => {
      await router.isReady();
      try {
        await Promise.all([
          useRoleStore().fetchRoleList(),
          useWorkspaceV1Store().fetchIamPolicy(),
        ]);
      } catch (error) {
        // Roles/IAM pre-fetch failed — proceed to render the wizard anyway.
        // The wizard's own RPC calls surface their errors; blocking setup
        // on a prefetch failure leaves the page stuck on a spinner.
        console.error("Setup prefetch failed:", error);
      }
      setReady(true);
    })();
  }, []);

  return (
    <>
      <ComponentPermissionGuard permissions={[...SETUP_PERMISSIONS]}>
        {ready ? (
          <SetupWizard />
        ) : (
          <div className="w-full h-full flex items-center justify-center">
            <Loader2 className="size-6 animate-spin" />
          </div>
        )}
      </ComponentPermissionGuard>
      <AuthFooter />
    </>
  );
}

function SetupWizard() {
  const { t } = useTranslation();
  const [currentStep, setCurrentStep] = useState(0);
  const [purpose, setPurpose] = useState<Purpose>("edit-schema");
  const [workflow, setWorkflow] = useState<Workflow>("team");
  const [data, setData] = useState<DataChoice>("self-setup");
  const [mode, setMode] = useState<DatabaseChangeMode>(
    DatabaseChangeMode.PIPELINE
  );
  const [projectTitle, setProjectTitle] = useState("New Project");
  const [resourceId, setResourceId] = useState("");
  const [loading, setLoading] = useState(false);

  const resourceFieldRef = useRef<ResourceIdFieldRef>(null);
  const [resourceValid, setResourceValid] = useState(false);

  const enableOnboarding = useVueState(
    () => useActuatorV1Store().enableOnboarding
  );
  const databaseChangeMode = useVueState(
    () => useAppFeature("bb.feature.database-change-mode").value
  );

  useEffect(() => {
    if (!enableOnboarding) {
      router.push(homePath(databaseChangeMode));
    }
  }, [enableOnboarding, databaseChangeMode]);

  const steps = [
    t("setup.basic-info"),
    t("setup.self"),
    t("settings.general.workspace.default-landing-page.self"),
  ];

  const canAdvance = (() => {
    if (currentStep === 1 && data === "self-setup") {
      return !!projectTitle && resourceValid;
    }
    return !loading;
  })();

  const changeStepIndex = (next: number) => {
    if (next === steps.length - 1) {
      if (purpose === "query-data" && workflow === "simple") {
        setMode(DatabaseChangeMode.EDITOR);
      } else {
        setMode(DatabaseChangeMode.PIPELINE);
      }
    }
    setCurrentStep(next);
  };

  const projectV1Store = useProjectV1Store();

  const skip = () => router.push("/");

  const finish = async () => {
    if (loading) return;
    setLoading(true);
    try {
      if (data === "self-setup") {
        const project = { ...unknownProject(), title: projectTitle };
        await projectV1Store.createProject(project, resourceId);
      } else {
        await useActuatorV1Store().setupSample();
      }
      await useSettingV1Store().updateWorkspaceProfile({
        payload: { databaseChangeMode: mode },
        updateMask: create(FieldMaskSchema, {
          paths: ["value.workspace_profile.database_change_mode"],
        }),
      });
      router.push(homePath(mode));
    } finally {
      setLoading(false);
    }
  };

  const validateProjectResourceID = useCallback(
    async (id: string) => {
      try {
        await projectV1Store.getOrFetchProjectByName(
          `${projectNamePrefix}${id}`,
          true /* silent */
        );
        return [
          {
            type: "error" as const,
            message: t("resource-id.validation.duplicated", {
              resource: "project",
            }),
          },
        ];
      } catch (error) {
        if (getErrorCode(error) === Code.NotFound) {
          return [];
        }
        return [
          {
            type: "error" as const,
            message: extractGrpcErrorMessage(error),
          },
        ];
      }
    },
    [projectV1Store, t]
  );

  return (
    <div className="w-full mx-auto max-w-2xl py-6 px-4 flex flex-col gap-4">
      <div className="sticky top-0 bg-white z-10 pb-4 border-b border-control-border">
        <ol className="flex items-center gap-x-4 text-sm">
          {steps.map((label, i) => (
            <li
              key={label}
              className={
                i === currentStep
                  ? "text-accent font-medium"
                  : "text-control-light"
              }
            >
              {i + 1}. {label}
            </li>
          ))}
        </ol>
      </div>

      {currentStep === 0 && (
        <div className="w-full flex flex-col gap-6 py-4">
          <div className="flex flex-col gap-y-2">
            <p>{t("setup.purposes.self")}</p>
            <RadioGroup
              value={purpose}
              onValueChange={(v) => setPurpose(v as Purpose)}
              className="flex-col items-start gap-y-2"
            >
              <RadioGroupItem value="edit-schema">
                {t("setup.purposes.alter-schema")}
              </RadioGroupItem>
              <RadioGroupItem value="query-data">
                {t("setup.purposes.query-data")}
              </RadioGroupItem>
            </RadioGroup>
          </div>
          <div className="flex flex-col gap-y-2">
            <p>{t("setup.workflow.self")}</p>
            <RadioGroup
              value={workflow}
              onValueChange={(v) => setWorkflow(v as Workflow)}
              className="flex-col items-start gap-y-2"
            >
              <RadioGroupItem value="team">
                {t("setup.workflow.team")}
              </RadioGroupItem>
              <RadioGroupItem value="simple">
                {t("setup.workflow.simple")}
              </RadioGroupItem>
            </RadioGroup>
          </div>
        </div>
      )}

      {currentStep === 1 && (
        <div className="w-full flex flex-col gap-2 py-4">
          <RadioGroup
            value={data}
            onValueChange={(v) => setData(v as DataChoice)}
            className="flex-col items-start gap-y-6"
          >
            <div>
              <RadioGroupItem value="self-setup">
                <div className="font-medium">{t("setup.data.self-setup")}</div>
              </RadioGroupItem>
              {data === "self-setup" && (
                <div className="mt-2 ml-6 flex flex-col gap-y-1">
                  <Input
                    value={projectTitle}
                    required
                    placeholder={t("project.create-modal.project-name")}
                    onChange={(e) => setProjectTitle(e.target.value)}
                  />
                  <ResourceIdField
                    suffix
                    ref={resourceFieldRef}
                    value={resourceId}
                    resourceName={t("common.project")}
                    resourceTitle={projectTitle}
                    validate={validateProjectResourceID}
                    onChange={setResourceId}
                    onValidationChange={setResourceValid}
                  />
                </div>
              )}
            </div>
            <div>
              <RadioGroupItem value="builtin-sample">
                <div className="flex flex-col gap-1">
                  <div className="font-medium">{t("setup.data.built-in")}</div>
                  <div>{t("setup.data.built-in-desc")}</div>
                </div>
              </RadioGroupItem>
            </div>
          </RadioGroup>
        </div>
      )}

      {currentStep === 2 && (
        <div className="py-4 flex flex-col gap-6">
          <div className="font-medium">
            {t("settings.general.workspace.default-landing-page.self")}
          </div>
          <RadioGroup
            value={String(mode)}
            onValueChange={(v) => setMode(Number(v) as DatabaseChangeMode)}
            className="flex-col items-start gap-y-6"
          >
            <RadioGroupItem value={String(DatabaseChangeMode.PIPELINE)}>
              <div className="flex flex-col gap-1">
                <div className="textinfo">
                  {t(
                    "settings.general.workspace.default-landing-page.workspace.self"
                  )}
                </div>
                <div className="textinfolabel">
                  {t(
                    "settings.general.workspace.default-landing-page.workspace.description"
                  )}
                </div>
              </div>
            </RadioGroupItem>
            <RadioGroupItem value={String(DatabaseChangeMode.EDITOR)}>
              <div className="flex flex-col gap-1">
                <div className="textinfo">
                  {t(
                    "settings.general.workspace.default-landing-page.sql-editor.self"
                  )}
                </div>
                <div className="textinfolabel">
                  {t(
                    "settings.general.workspace.default-landing-page.sql-editor.description"
                  )}
                </div>
              </div>
            </RadioGroupItem>
          </RadioGroup>
        </div>
      )}

      <div className="flex items-center justify-between pt-4 border-t border-control-border">
        <Button variant="ghost" onClick={skip}>
          {t("setup.skip-setup")}
        </Button>
        <div className="flex items-center gap-x-2">
          {currentStep > 0 && (
            <Button
              variant="outline"
              onClick={() => changeStepIndex(currentStep - 1)}
            >
              &larr;
            </Button>
          )}
          {currentStep < steps.length - 1 ? (
            <Button
              onClick={() => changeStepIndex(currentStep + 1)}
              disabled={!canAdvance}
            >
              &rarr;
            </Button>
          ) : (
            <Button onClick={finish} disabled={loading}>
              {t("common.confirm")}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
