import type { TFunction } from "i18next";
import { Circle, CircleAlert, TriangleAlert } from "lucide-react";
import { useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { InlineCode } from "@/components/WorkloadIdentityFlow/StepParts";
import { cn } from "@/lib/utils";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import {
  EXTERNAL_URL_PLACEHOLDER,
  GITLAB_INCLUDE,
  type WorkflowFile,
  type WorkflowProvider,
} from "../gitopsWorkflowYaml";
import { ExportSchemaField } from "./ExportSchemaField";
import { ProviderLogo } from "./ProviderLogo";
import { SchemaFiles } from "./SchemaFiles";
import { goToSetupField, type SetupFieldKey } from "./setupField";
import type { SchemaExportState } from "./useSchemaExport";

// A file with more lines than this starts cut here, with an action that shows
// the rest, so the steps after it stay close.
export const FILE_PREVIEW_LINES = 12;

// A choice setup still needs before it writes any file.
export interface OpenChoice {
  key: SetupFieldKey;
  label: string;
  // Picked, but setup cannot use it as it is: what it needs.
  problem?: string;
}

// What the steps show once no choice is open.
export interface FileSteps {
  folder: string;
  // Shown only when the workload identity names one repository.
  repositoryName: string;
  repositoryUrl: string;
  files: WorkflowFile[];
  // The External URL is not set, so the files hold a placeholder.
  urlMissing: boolean;
  // State-based only. The schema files step can be done later, so no database
  // may be picked yet.
  exportCandidates: Database[];
  schemaDatabase: Database | undefined;
  schemaExport: SchemaExportState | undefined;
  onSchemaDatabaseChange: (name: string) => void;
}

// The second half of the page: the files to add to the repository, as numbered
// steps. While a choice is open it shows no file: the steps are locked, under
// the open choices.
export function FilesSection({
  declarative,
  provider,
  waitingFor,
  steps,
}: {
  readonly declarative: boolean;
  // Undefined until a workload identity for a git provider is picked.
  readonly provider: WorkflowProvider | undefined;
  readonly waitingFor: OpenChoice[];
  // Undefined while a choice is open.
  readonly steps: FileSteps | undefined;
}) {
  const { t } = useTranslation();
  return (
    <section
      aria-labelledby="gitops-files-heading"
      className="flex flex-col gap-y-4"
    >
      <div className="flex flex-col gap-y-1">
        <h2
          id="gitops-files-heading"
          className="text-base font-semibold text-main"
        >
          {t("gitops.setup.files.heading")}
        </h2>
        <p className="text-sm text-pretty text-control-light">
          {steps
            ? declarative
              ? t("gitops.setup.files.description-state")
              : t("gitops.setup.files.description")
            : declarative
              ? t("gitops.setup.files.description-pending-state")
              : t("gitops.setup.files.description-pending")}
        </p>
      </div>
      {steps && provider ? (
        <Steps declarative={declarative} provider={provider} {...steps} />
      ) : (
        <LockedSteps
          declarative={declarative}
          provider={provider}
          waitingFor={waitingFor}
        />
      )}
    </section>
  );
}

// The last step is named for the git provider, once the identity tells it.
const requestStepTitle = (
  t: TFunction,
  provider: WorkflowProvider | undefined
) =>
  provider === "gitlab"
    ? t("gitops.setup.files.request-step-title-gitlab")
    : provider === "github"
      ? t("gitops.setup.files.request-step-title-github")
      : t("gitops.setup.files.request-step-title");

function LockedSteps({
  declarative,
  provider,
  waitingFor,
}: {
  readonly declarative: boolean;
  readonly provider: WorkflowProvider | undefined;
  readonly waitingFor: OpenChoice[];
}) {
  const { t } = useTranslation();
  return (
    <>
      <div
        role="group"
        aria-labelledby="gitops-waiting-for"
        className="flex flex-wrap items-center gap-2"
      >
        <span id="gitops-waiting-for" className="text-sm text-control-light">
          {t("gitops.setup.files.waiting-for")}
        </span>
        {waitingFor.map((choice) => (
          <Button
            key={choice.key}
            appearance="outline"
            data-choice={choice.key}
            className="h-auto min-h-9 rounded-full border-dashed py-1.5 whitespace-normal"
            onClick={() => goToSetupField(choice.key)}
          >
            {choice.problem ? (
              <CircleAlert
                aria-hidden="true"
                className="size-4 shrink-0 text-warning"
              />
            ) : (
              <Circle
                aria-hidden="true"
                className="size-4 shrink-0 text-control-placeholder"
              />
            )}
            {choice.label}
            {choice.problem && (
              <span className="font-normal text-warning">{choice.problem}</span>
            )}
          </Button>
        ))}
      </div>
      <ol data-locked="" className="flex flex-col gap-y-4">
        <GuideStep
          locked
          number={1}
          title={t("gitops.setup.files.workflow-step-title")}
          description={t("gitops.setup.files.locked-workflow-step")}
        />
        {declarative && (
          <GuideStep
            locked
            number={2}
            title={t("gitops.setup.files.schema-step-title")}
            description={t("gitops.setup.files.locked-schema-step")}
          />
        )}
        <GuideStep
          locked
          number={declarative ? 3 : 2}
          title={requestStepTitle(t, provider)}
          description={t("gitops.setup.files.locked-request-step")}
        />
      </ol>
    </>
  );
}

function Steps({
  provider,
  declarative,
  folder,
  repositoryName,
  repositoryUrl,
  files,
  urlMissing,
  exportCandidates,
  schemaDatabase,
  schemaExport,
  onSchemaDatabaseChange,
}: FileSteps & {
  readonly provider: WorkflowProvider;
  readonly declarative: boolean;
}) {
  const { t } = useTranslation();
  const isGitLab = provider === "gitlab";
  const exampleFile = `${folder}/0001_create_orders.sql`;

  return (
    <>
      {repositoryName && repositoryUrl && (
        <p className="flex min-w-0 flex-wrap items-center gap-x-2 text-sm">
          <span className="text-control-light">
            {t("gitops.setup.files.repository")}
          </span>
          <a
            href={repositoryUrl}
            target="_blank"
            rel="noreferrer"
            className="inline-flex max-w-full items-center gap-x-1.5 text-accent hover:underline"
          >
            <ProviderLogo provider={provider} />
            <span className="min-w-0 wrap-anywhere">{repositoryName}</span>
          </a>
        </p>
      )}

      <ol className="flex flex-col gap-y-6">
        <GuideStep
          number={1}
          title={t("gitops.setup.files.workflow-step-title")}
          description={t("gitops.setup.files.workflow-step-description")}
        >
          {urlMissing && (
            <p className="flex items-start gap-x-2 text-sm text-warning">
              <TriangleAlert className="mt-0.5 size-4 shrink-0" />
              <span>
                <Trans
                  i18nKey="gitops.setup.files.url-missing"
                  values={{ placeholder: EXTERNAL_URL_PLACEHOLDER }}
                  components={{ code: <PlaceholderMark /> }}
                />
              </span>
            </p>
          )}
          {files.map((file) => (
            <FileBlock
              key={file.path}
              path={file.path}
              content={file.content}
              copyOff={urlMissing}
            />
          ))}
          {isGitLab && (
            <div className="flex flex-col gap-y-1.5">
              <p className="text-sm text-control-light">
                <Trans
                  i18nKey="gitops.setup.files.gitlab-include-note"
                  components={{ code: <InlineCode /> }}
                />
              </p>
              <FileBlock
                path=".gitlab-ci.yml"
                content={GITLAB_INCLUDE}
                copyOff={false}
              />
            </div>
          )}
        </GuideStep>

        {declarative && (
          <GuideStep
            number={2}
            title={t("gitops.setup.files.schema-step-title")}
            description={
              <Trans
                i18nKey="gitops.setup.files.schema-step-description"
                values={{ folder: `${folder}/` }}
                components={{ code: <InlineCode /> }}
              />
            }
          >
            <ExportSchemaField
              candidates={exportCandidates}
              value={schemaDatabase}
              onChange={onSchemaDatabaseChange}
            />
            {schemaDatabase && schemaExport && (
              // Keyed so the filter starts empty for another database.
              <SchemaFiles
                key={schemaDatabase.name}
                database={schemaDatabase}
                folder={folder}
                state={schemaExport}
              />
            )}
          </GuideStep>
        )}

        <GuideStep
          number={declarative ? 3 : 2}
          title={requestStepTitle(t, provider)}
          description={
            isGitLab
              ? t("gitops.setup.files.request-step-description-gitlab")
              : t("gitops.setup.files.request-step-description-github")
          }
        >
          <p className="text-sm text-pretty text-control-light">
            {declarative
              ? isGitLab
                ? t("gitops.setup.files.next-state-gitlab")
                : t("gitops.setup.files.next-state-github")
              : isGitLab
                ? t("gitops.setup.files.next-gitlab")
                : t("gitops.setup.files.next-github")}
          </p>
          <p className="text-sm text-pretty text-control-light">
            {declarative ? (
              isGitLab ? (
                t("gitops.setup.files.developers-state-gitlab")
              ) : (
                t("gitops.setup.files.developers-state-github")
              )
            ) : isGitLab ? (
              <Trans
                i18nKey="gitops.setup.files.developers-gitlab"
                values={{ file: exampleFile }}
                components={{ code: <InlineCode /> }}
              />
            ) : (
              <Trans
                i18nKey="gitops.setup.files.developers-github"
                values={{ file: exampleFile }}
                components={{ code: <InlineCode /> }}
              />
            )}
          </p>
        </GuideStep>
      </ol>
    </>
  );
}

// One numbered step. A locked one is dimmed and has no body.
function GuideStep({
  number,
  title,
  description,
  locked,
  children,
}: {
  readonly number: number;
  readonly title: React.ReactNode;
  readonly description: React.ReactNode;
  readonly locked?: boolean;
  readonly children?: React.ReactNode;
}) {
  return (
    <li className="flex min-w-0 gap-x-3">
      <span
        aria-hidden="true"
        className={cn(
          "flex size-6 shrink-0 items-center justify-center rounded-full border border-control-border text-xs font-medium tabular-nums",
          locked ? "border-dashed text-control-placeholder" : "text-control"
        )}
      >
        {number}
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-y-3">
        <div className={cn("flex flex-col pt-0.5", !locked && "gap-y-1")}>
          <h3
            className={cn(
              "text-sm font-medium",
              locked ? "text-control-light" : "text-main"
            )}
          >
            {title}
          </h3>
          <p className="text-sm text-pretty text-control-light">
            {description}
          </p>
        </div>
        {children}
      </div>
    </li>
  );
}

function PlaceholderMark({
  children,
}: {
  readonly children?: React.ReactNode;
}) {
  return (
    <mark className="rounded-xs bg-warning/20 px-1 font-mono text-warning">
      {children}
    </mark>
  );
}

// One file. A long one shows its first lines until the owner asks for all of
// them; Copy takes the whole file either way.
function FileBlock({
  path,
  content,
  copyOff,
}: {
  readonly path: string;
  readonly content: string;
  readonly copyOff: boolean;
}) {
  const { t } = useTranslation();
  const [showAll, setShowAll] = useState(false);
  const lines = content.replace(/\n$/, "").split("\n");
  const long = lines.length > FILE_PREVIEW_LINES;
  const cut = long && !showAll;
  const shown = cut ? lines.slice(0, FILE_PREVIEW_LINES).join("\n") : content;
  // Each part keyed by where it starts in the shown text.
  let start = 0;
  const parts = shown.split(EXTERNAL_URL_PLACEHOLDER).map((text) => {
    const part = { start, text };
    start += text.length + EXTERNAL_URL_PLACEHOLDER.length;
    return part;
  });
  return (
    <div className="overflow-hidden rounded-sm border border-block-border">
      <div
        data-slot="code-block-header"
        className="flex items-center justify-between gap-x-2 border-b border-block-border bg-background py-1.5 pr-2 pl-3"
      >
        <div className="flex min-w-0 items-center gap-x-1">
          <span className="min-w-0 font-mono text-xs wrap-anywhere text-main">
            {path}
          </span>
          <CopyButton
            content={path}
            label={t("gitops.setup.files.copy-path")}
            className="shrink-0"
          />
        </div>
        <CopyButton
          content={content}
          size="sm"
          appearance="solid"
          disabled={copyOff}
          label={copyOff ? t("gitops.setup.files.copy-off") : undefined}
          className="shrink-0"
        >
          {t("common.copy")}
        </CopyButton>
      </div>
      <div className="relative">
        <pre className="overflow-x-auto bg-control-bg p-3 font-mono text-xs whitespace-pre text-main">
          {parts.map((part) => (
            <span key={part.start}>
              {part.start > 0 && (
                <PlaceholderMark>{EXTERNAL_URL_PLACEHOLDER}</PlaceholderMark>
              )}
              {part.text}
            </span>
          ))}
        </pre>
        {cut && (
          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-x-0 bottom-0 h-8 bg-gradient-to-t from-control-bg to-transparent"
          />
        )}
      </div>
      {long && (
        <div
          data-slot="code-block-footer"
          className="border-t border-block-border bg-background px-1 py-1"
        >
          <Button
            appearance="link"
            size="xs"
            aria-expanded={showAll}
            onClick={() => setShowAll((value) => !value)}
          >
            {showAll
              ? t("common.show-less")
              : t("gitops.setup.files.show-all", { lines: lines.length })}
          </Button>
        </div>
      )}
    </div>
  );
}
