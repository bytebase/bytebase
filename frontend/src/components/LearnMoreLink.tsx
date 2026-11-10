import { ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";

interface LearnMoreLinkProps {
  href: string;
}

export function LearnMoreLink({ href }: LearnMoreLinkProps) {
  const { t } = useTranslation();
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-x-1 underline"
    >
      {t("common.learn-more")}
      <ExternalLink className="w-3 h-3" />
    </a>
  );
}
