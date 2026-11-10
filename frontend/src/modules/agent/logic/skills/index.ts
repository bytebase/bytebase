import { skill as databaseChangeSkill } from "./databaseChange";
import { maskingSections, skill as dataMaskingSkill } from "./dataMasking";
import { skill as grantPermissionSkill } from "./grantPermission";
import { skill as querySkill } from "./query";

interface Skill {
  name: string;
  description: string;
  content: string;
}

const skills: Skill[] = [
  querySkill,
  databaseChangeSkill,
  grantPermissionSkill,
  dataMaskingSkill,
];

const skillMap = new Map(skills.map((s) => [s.name, s]));

export interface GetSkillArgs {
  name?: string;
  section?: MaskingSkillSection;
}

export type MaskingSkillSection = keyof typeof maskingSections;

export type SkillLoadResult =
  | {
      kind: "loaded";
      name: string;
      section?: MaskingSkillSection;
      content: string;
    }
  | { kind: "listed"; content: string }
  | { kind: "error"; message: string };

export function getMaskingSkillGuidance(
  sections: readonly MaskingSkillSection[]
): string {
  if (sections.length === 0) return "";
  const active = new Set<MaskingSkillSection>(["overview", ...sections]);
  return (Object.keys(maskingSections) as MaskingSkillSection[])
    .filter((section) => active.has(section))
    .map((section) => maskingSections[section])
    .join("\n\n");
}

export function resolveSkill(args: GetSkillArgs): SkillLoadResult {
  if (!args.name) {
    if (args.section !== undefined) {
      return {
        kind: "error",
        message: "A skill name is required with section.",
      };
    }
    const list = skills
      .map((s) => `- **${s.name}**: ${s.description}`)
      .join("\n");
    return {
      kind: "listed",
      content: `Available skills:\n${list}\n\nCall get_skill(name="skill-name") to load a specific skill. For data-masking, section defaults to overview; use rule, exemption, or semantic-type for that workflow.`,
    };
  }

  const skill = skillMap.get(args.name);
  if (!skill) {
    const names = skills.map((s) => s.name).join(", ");
    return {
      kind: "error",
      message: `Skill "${args.name}" not found. Available skills: ${names}`,
    };
  }

  if (skill.name === dataMaskingSkill.name) {
    const section = args.section === undefined ? "overview" : args.section;
    if (
      typeof section !== "string" ||
      !Object.hasOwn(maskingSections, section)
    ) {
      return {
        kind: "error",
        message: `Unknown data-masking section "${section}". Available sections: ${Object.keys(maskingSections).join(", ")}`,
      };
    }
    return {
      kind: "loaded",
      name: skill.name,
      section,
      content: getMaskingSkillGuidance([section]),
    };
  }
  if (args.section !== undefined) {
    return {
      kind: "error",
      message: `Skill "${skill.name}" does not support sections.`,
    };
  }
  return { kind: "loaded", name: skill.name, content: skill.content };
}

export function getSkill(args: GetSkillArgs): string {
  const result = resolveSkill(args);
  return result.kind === "error" ? result.message : result.content;
}
