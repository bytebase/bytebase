import { useState } from "react";
import { useTranslation } from "react-i18next";
import { FormError, FormField, FormLabel } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { isValidMigrationFolder } from "../gitopsWorkflowYaml";

// "./db/migrations/" and "db/migrations" name the same folder.
const normalizeFolder = (value: string) =>
  value.trim().replace(/^\.\//, "").replace(/\/+$/, "");

// The folder the workflow watches. `value` is the last valid folder, which
// follows the workflow type until the owner types. The field keeps what is
// typed, so an invalid entry stays visible while the files stay complete.
export function FolderField({
  declarative,
  value,
  onChange,
}: {
  readonly declarative: boolean;
  readonly value: string;
  readonly onChange: (folder: string) => void;
}) {
  const { t } = useTranslation();
  const [typed, setTyped] = useState<string>();
  const draft = typed ?? value;
  const valid = isValidMigrationFolder(normalizeFolder(draft));
  const change = (text: string) => {
    setTyped(text);
    const folder = normalizeFolder(text);
    if (isValidMigrationFolder(folder)) onChange(folder);
  };
  return (
    <FormField>
      <FormLabel htmlFor="gitops-folder">
        {declarative
          ? t("gitops.setup.files.schema-folder-label")
          : t("gitops.setup.files.folder-label")}
      </FormLabel>
      <Input
        id="gitops-folder"
        className="max-w-80 font-mono"
        value={draft}
        spellCheck={false}
        autoComplete="off"
        autoCapitalize="off"
        aria-invalid={!valid}
        onChange={(event) => change(event.target.value)}
      />
      {valid ? (
        <p className="text-xs text-control-light">
          {declarative
            ? t("gitops.setup.files.schema-folder-hint")
            : t("gitops.setup.files.folder-hint")}
        </p>
      ) : (
        <FormError>
          {declarative
            ? t("gitops.setup.files.schema-folder-invalid")
            : t("gitops.setup.files.folder-invalid")}
        </FormError>
      )}
    </FormField>
  );
}
