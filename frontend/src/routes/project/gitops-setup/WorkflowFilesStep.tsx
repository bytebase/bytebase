import { TriangleAlert } from "lucide-react";
import { useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { Input } from "@/components/ui/input";
import {
  Field,
  InlineCode,
  StepHeading,
  TextAction,
} from "@/components/WorkloadIdentityFlow/StepParts";
import {
  DEFAULT_MIGRATION_FOLDER,
  EXTERNAL_URL_PLACEHOLDER,
  GITLAB_INCLUDE,
  isValidMigrationFolder,
  type WorkflowFile,
  type WorkflowProvider,
} from "../gitopsWorkflowYaml";
import { ProviderLogo } from "./ProviderLogo";

// A file with more lines than this starts cut here, with an action that shows
// the rest. The step is then short enough to show which files there are.
export const FILE_PREVIEW_LINES = 12;

// "./db/migrations/" and "db/migrations" name the same folder.
export const normalizeMigrationFolder = (value: string) =>
  value.trim().replace(/^\.\//, "").replace(/\/+$/, "");

// Step 3: the files to add to the repository, built from steps 1 and 2.
export function WorkflowFilesStep({
  provider,
  folder,
  onFolderChange,
  repositoryName,
  repositoryUrl,
  files,
  urlMissing,
}: {
  readonly provider: WorkflowProvider;
  readonly folder: string;
  readonly onFolderChange: (folder: string) => void;
  // Shown only when the workload identity names one repository.
  readonly repositoryName: string;
  readonly repositoryUrl: string;
  readonly files: WorkflowFile[];
  // The External URL is not set, so the files hold a placeholder.
  readonly urlMissing: boolean;
}) {
  const { t } = useTranslation();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(folder);
  const draftValid = isValidMigrationFolder(normalizeMigrationFolder(draft));

  const changeDraft = (value: string) => {
    setDraft(value);
    const normalized = normalizeMigrationFolder(value);
    if (isValidMigrationFolder(normalized)) onFolderChange(normalized);
  };
  const reset = () => {
    setDraft(DEFAULT_MIGRATION_FOLDER);
    onFolderChange(DEFAULT_MIGRATION_FOLDER);
    setEditing(false);
  };
  const isGitLab = provider === "gitlab";
  const exampleFile = `${folder}/0001_create_orders.sql`;

  return (
    <div className="flex flex-col gap-y-4">
      <StepHeading
        question={t("gitops.setup.files.heading")}
        description={t("gitops.setup.files.description")}
      />

      <Field
        id="gitops-migration-folder"
        label={t("gitops.setup.files.folder-label")}
        action={
          editing ? (
            <TextAction onClick={reset}>{t("common.reset")}</TextAction>
          ) : (
            <TextAction
              onClick={() => {
                setDraft(folder);
                setEditing(true);
              }}
            >
              {t("common.edit")}
            </TextAction>
          )
        }
        hint={t("gitops.setup.files.folder-hint")}
        error={
          editing && !draftValid
            ? t("gitops.setup.files.folder-invalid")
            : undefined
        }
      >
        {editing ? (
          <Input
            id="gitops-migration-folder"
            className="font-mono"
            value={draft}
            onChange={(event) => changeDraft(event.target.value)}
          />
        ) : (
          <div className="rounded-sm border border-block-border bg-control-bg/50 px-3 py-2 font-mono text-sm text-main">
            {folder}/
          </div>
        )}
      </Field>

      {repositoryName && repositoryUrl && (
        <Field label={t("gitops.setup.files.repository")}>
          <a
            href={repositoryUrl}
            target="_blank"
            rel="noreferrer"
            className="inline-flex max-w-full items-center gap-x-1.5 self-start text-sm text-accent hover:underline"
          >
            <ProviderLogo provider={provider} />
            <span className="min-w-0 wrap-anywhere">{repositoryName}</span>
          </a>
        </Field>
      )}

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
        <FileBlock
          path=".gitlab-ci.yml"
          content={GITLAB_INCLUDE}
          copyOff={false}
        />
      )}

      <section className="flex flex-col gap-y-2 border-t border-block-border pt-4">
        <h4 className="text-sm font-medium text-main">
          {t("gitops.setup.files.next-title")}
        </h4>
        <p className="text-sm text-control-light">
          {isGitLab
            ? t("gitops.setup.files.next-gitlab")
            : t("gitops.setup.files.next-github")}
        </p>
        <p className="text-sm text-control-light">
          {isGitLab ? (
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
      </section>
    </div>
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

// One file: its path in a header bar with a copy button, then the
// content, with the External URL placeholder marked. A long file shows its
// first lines until the owner asks for all of them. Copy takes the whole file
// either way.
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
        <span className="min-w-0 font-mono text-xs wrap-anywhere text-main">
          {path}
        </span>
        <div className="shrink-0">
          <CopyButton
            content={content}
            size="sm"
            disabled={copyOff}
            label={copyOff ? t("gitops.setup.files.copy-off") : undefined}
          />
        </div>
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
