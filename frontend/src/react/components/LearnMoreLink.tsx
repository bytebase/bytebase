import { ExternalLink } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/react/lib/utils";

interface LearnMoreLinkProps {
  href: string;
  className?: string;
}

export function LearnMoreLink({ href, className }: LearnMoreLinkProps) {
  const { t } = useTranslation();
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className={cn(
        "inline-flex items-center gap-x-0.5 hover:underline",
        className
      )}
    >
      {t("common.learn-more")}
      <ExternalLink className="w-3 h-3" />
    </a>
  );
}
