import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  FormField,
  FormFieldGroup,
  FormLabel,
  FormTitle,
} from "@/components/ui/form";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { isGuideTeamSize } from "@/modules/workspace-setup-guide/selection";
import {
  GUIDE_TEAM_SIZES,
  type GuideTeamSize,
} from "@/modules/workspace-setup-guide/types";
import type { SetupSource } from "./draft";
import type { SetupFinishStage } from "./finish";
import { InviteEmailsInput } from "./InviteEmailsInput";
import { describeInviteAddresses, validInviteAddresses } from "./inviteEmails";
import { SetupNote, SetupScreenFrame } from "./SetupScreenFrame";

export type SetupFinishFailure = {
  stage: SetupFinishStage;
  detail?: string;
};

export function TeamScreen({
  teamSize,
  addresses,
  ownerEmail,
  projectTitle,
  source,
  isSaaS,
  finishing,
  failure,
  onTeamSizeChange,
  onAddressesChange,
  onBack,
  onFinish,
  onFinishWithoutSample,
}: {
  readonly teamSize: GuideTeamSize | undefined;
  readonly addresses: readonly string[];
  readonly ownerEmail: string;
  readonly projectTitle: string;
  readonly source: SetupSource;
  // An invitation is an email, and only Cloud can send one at first run.
  // Self-hosted lists no one: a member added there would have no password.
  readonly isSaaS: boolean;
  readonly finishing: boolean;
  readonly failure: SetupFinishFailure | undefined;
  readonly onTeamSizeChange: (size: GuideTeamSize) => void;
  readonly onAddressesChange: (addresses: string[]) => void;
  readonly onBack: () => void;
  readonly onFinish: () => void;
  readonly onFinishWithoutSample: () => void;
}) {
  const { t } = useTranslation();
  // The list as it was when the button was pressed. Text left in the field is
  // entered as the press takes the focus; a press that makes it an entry that
  // is not an address stops there, so that entry is seen before Finish runs.
  const listAtPress = useRef<readonly string[] | undefined>(undefined);
  const pressed = () => {
    listAtPress.current = addresses;
  };
  const enteredNonAddress = () => {
    const before = listAtPress.current;
    listAtPress.current = undefined;
    return describeInviteAddresses(
      addresses.filter((address) => before && !before.includes(address)),
      ownerEmail
    ).some(({ problem }) => problem === "invalid");
  };
  const invites = isSaaS && !!teamSize && teamSize !== "just-me";
  const inviteeCount = invites
    ? validInviteAddresses(addresses, ownerEmail).length
    : 0;

  const sizeLabel = (size: GuideTeamSize) => {
    switch (size) {
      case "just-me":
        return t("workspace-setup.team.sizes.just-me");
      case "2-10":
        return t("workspace-setup.team.sizes.2-10");
      case "11-50":
        return t("workspace-setup.team.sizes.11-50");
      case "51-200":
        return t("workspace-setup.team.sizes.51-200");
      case "200+":
        return t("workspace-setup.team.sizes.200-plus");
    }
  };
  const failureTitle = (stage: SetupFinishStage) => {
    switch (stage) {
      case "names":
        return t("workspace-setup.team.failed.names");
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
  const finishLabel = finishing
    ? t("workspace-setup.team.finishing")
    : failure
      ? t("workspace-setup.team.try-again")
      : inviteeCount === 0
        ? t("workspace-setup.team.finish")
        : t("workspace-setup.team.finish-invite", { count: inviteeCount });

  return (
    <SetupScreenFrame
      screen="team"
      title={t("workspace-setup.team.title")}
      description={t("workspace-setup.team.description")}
      onSubmit={() => {
        if (enteredNonAddress()) return;
        if (teamSize && !finishing) onFinish();
      }}
      footer={
        <>
          <Button
            type="button"
            appearance="secondary"
            disabled={finishing}
            onClick={onBack}
          >
            {t("common.back")}
          </Button>
          <div className="flex flex-wrap items-center justify-end gap-2">
            {failure?.stage === "sample" && (
              <Button
                type="button"
                appearance="secondary"
                onPointerDown={pressed}
                onClick={() => {
                  if (!enteredNonAddress()) onFinishWithoutSample();
                }}
              >
                {t("workspace-setup.team.finish-without-sample")}
              </Button>
            )}
            <Button
              type="submit"
              disabled={!teamSize || finishing}
              onPointerDown={pressed}
            >
              {finishLabel}
            </Button>
          </div>
        </>
      }
    >
      <FormFieldGroup>
        <FormField>
          <FormTitle>{t("workspace-setup.team.size")}</FormTitle>
          <RadioGroup
            aria-label={t("workspace-setup.team.size")}
            aria-describedby={
              teamSize ? undefined : "workspace-setup-team-size-note"
            }
            value={teamSize ?? ""}
            disabled={finishing}
            onValueChange={(value) => {
              if (isGuideTeamSize(value)) onTeamSizeChange(value);
            }}
            className="flex-wrap gap-y-2"
          >
            {GUIDE_TEAM_SIZES.map((size) => (
              <RadioGroupItem key={size} value={size}>
                {sizeLabel(size)}
              </RadioGroupItem>
            ))}
          </RadioGroup>
          {!teamSize && (
            <SetupNote id="workspace-setup-team-size-note">
              {t("workspace-setup.team.size-required")}
            </SetupNote>
          )}
        </FormField>
        {invites && (
          <FormField>
            <FormLabel htmlFor="workspace-setup-invitees">
              {t("workspace-setup.team.invite-teammates")}
              <span className="ml-1 font-normal text-control-light">
                {t("workspace-setup.optional")}
              </span>
            </FormLabel>
            <InviteEmailsInput
              id="workspace-setup-invitees"
              addresses={addresses}
              ownerEmail={ownerEmail}
              describedBy="workspace-setup-invitees-note"
              disabled={finishing}
              onChange={onAddressesChange}
            />
            <SetupNote id="workspace-setup-invitees-note">
              {[
                t("workspace-setup.team.members-enter"),
                t("workspace-setup.team.members-role", {
                  project: projectTitle,
                }),
                source === "own" && t("workspace-setup.team.members-own"),
                t("workspace-setup.team.members-roles-later"),
              ]
                .filter(Boolean)
                .join(" ")}
            </SetupNote>
          </FormField>
        )}
      </FormFieldGroup>
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
    </SetupScreenFrame>
  );
}
