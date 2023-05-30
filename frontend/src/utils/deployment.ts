import { PRESET_DB_NAME_TEMPLATE_PLACEHOLDERS } from "./label";

export const buildDatabaseNameRegExpByTemplate = (template: string): RegExp => {
  let regexpString = template;

  /*
    Rewrite the placeholder-based template to a big RegExp
    e.g. template = "{{DB_NAME}}__{{TENANT}}"
    here regex will be /^(?<DB_NAME>.+?)__(?<TENANT>.+?)$/
  */
  PRESET_DB_NAME_TEMPLATE_PLACEHOLDERS.forEach((placeholder) => {
    const pattern = `{{${placeholder}}}`;
    const groupRegExp = `(?<${placeholder}>.+?)`;
    regexpString = regexpString.replace(pattern, groupRegExp);
  });

  const regexp = new RegExp(`^${regexpString}$`);
  return regexp;
};

export const parseDatabaseLabelValueByTemplate = (
  template: string,
  name: string,
  group: "DB_NAME" | "TENANT"
) => {
  if (!template) return "";

  const regex = buildDatabaseNameRegExpByTemplate(template);
  const matches = name.match(regex);
  if (!matches) return "";

  const value = matches.groups?.[group];
  if (!value) return "";
  return value;
};
