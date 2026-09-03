import type { ReactElement } from "react";
import { useTranslation } from "react-i18next";
import { MarkdownEditor } from "@/components/MarkdownEditor";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

const guideFiles = import.meta.glob<string>(
  "../locales/guides/how-bytebase-works.*.md",
  { eager: true, import: "default", query: "?raw" }
);

export function getHowBytebaseWorksGuideContent(
  locale: string
): string | undefined {
  const content =
    guideFiles[`../locales/guides/how-bytebase-works.${locale}.md`];
  return content?.trim() ? content : undefined;
}

export function HowBytebaseWorksSheet({
  open,
  onOpenChange,
}: Props): ReactElement | null {
  const { i18n, t } = useTranslation();
  const guideContent = getHowBytebaseWorksGuideContent(
    i18n.resolvedLanguage ?? "en-US"
  );

  if (!guideContent) {
    return null;
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent width="narrow">
        <SheetHeader>
          <SheetTitle>{t("workspace-setup-guide.product-model")}</SheetTitle>
        </SheetHeader>
        <SheetBody>
          <MarkdownEditor content={guideContent} mode="preview" />
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}
