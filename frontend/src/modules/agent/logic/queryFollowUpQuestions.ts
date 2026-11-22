import enUS from "@/locales/en-US.json";
import esES from "@/locales/es-ES.json";
import jaJP from "@/locales/ja-JP.json";
import viVN from "@/locales/vi-VN.json";
import zhCN from "@/locales/zh-CN.json";

const templates = [enUS, esES, jaJP, viVN, zhCN]
  .flatMap(({ agent }) => [
    agent["follow-up"]["query-mask"],
    agent["follow-up"]["query-mask-database"],
    agent["follow-up"]["query-unmask"],
    agent["follow-up"]["query-unmask-database"],
  ])
  .map((template) => template.toLowerCase().split("{{database}}"));

export function isCanonicalQueryMaskingQuestion(prompt: string): boolean {
  if (/[\r\n]/.test(prompt)) return false;
  const question = prompt.trim().toLowerCase();
  return templates.some(([prefix, suffix]) => {
    if (suffix === undefined) return question === prefix;
    return (
      question.startsWith(prefix) &&
      question.endsWith(suffix) &&
      question.slice(prefix.length, question.length - suffix.length).trim()
        .length > 0
    );
  });
}
