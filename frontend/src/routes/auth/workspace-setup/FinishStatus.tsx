import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { useAppStore } from "@/stores/app";
import type { SetupSource } from "./draft";
import type { SetupFinishStage } from "./finish";

export type SetupFinishFailure = {
  stage: SetupFinishStage;
  detail?: string;
};

// What Finish is doing, or why it failed, on whichever step holds Finish: the
// team step on Cloud, the project step on self-hosted.
export function FinishStatus({
  finishing,
  failure,
  projectTitle,
  source,
}: {
  readonly finishing: boolean;
  readonly failure: SetupFinishFailure | undefined;
  readonly projectTitle: string;
  readonly source: SetupSource;
}) {
  const { t } = useTranslation();
  // Self-hosted saves only the workspace name; the person named themselves at
  // sign-up.
  const isSaaS = useAppStore((state) => state.isSaaSMode());
  const failureTitle = (stage: SetupFinishStage) => {
    switch (stage) {
      case "names":
        return isSaaS
          ? t("workspace-setup.team.failed.names")
          : t("workspace-setup.team.failed.workspace-name");
      case "project":
        return t("workspace-setup.team.failed.project", {
          project: projectTitle,
        });
      case "sample":
        return t("workspace-setup.team.failed.sample");
      case "invitees":
        return t("workspace-setup.team.failed.invitees");
    }
  };

  return (
    <>
      {failure && (
        <Alert
          variant="error"
          title={failureTitle(failure.stage)}
          description={failure.detail}
        />
      )}
      {finishing && (
        <p role="status" className="text-sm text-control-light">
          {source === "sample"
            ? t("workspace-setup.team.progress-sample", {
                project: projectTitle,
              })
            : t("workspace-setup.team.progress", { project: projectTitle })}
        </p>
      )}
    </>
  );
}
