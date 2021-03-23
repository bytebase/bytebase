export type LinkType =
  | "UNKNOWN"
  | "USER"
  | "TASK"
  | "ENVIRONMENT"
  | "INSTANCE"
  | "DATABASE"
  | "DATASOURCE"
  | "BOOKMARK";

export function validLink(link: string): boolean {
  return link != undefined;
}

export function linkfy(link: string): string {
  if (!link.startsWith("/")) {
    return "/" + link;
  }
  return link;
}
