import { Template, TemplateInput, InputType } from "./types";

const TEMPLATE_BRACKET_LEFT = "{{";
const TEMPLATE_BRACKET_RIGHT = "}}";

// getTemplateInputs will convert the string value into TemplateInput array.
// For example:
// "abc{{template}}" -> [{value: "abc", type: "string"}, {value: "template", type: "template"}]
// "abc{{not_template}}{{template}}" -> [{value: "abc{{not_template}}", type: "string"}, {value: "template", type: "template"}]
// "abc{{not_template}}{{template}}}}" -> [{value: "abc{{not_template}}", type: "string"}, {value: "template", type: "template"}, {value: "}}", type: "string"}]
// "{{abc{{}}{{template}}}}" -> [{value: "{{abc{{}}", type: "string"}, {value: "template", type: "template"}, {value: "}}", type: "string"}]
export const getTemplateInputs = (
  value: string,
  templates: Template[]
): TemplateInput[] => {
  let start = 0;
  let end = 0;
  const res: TemplateInput[] = [];
  const templateSet = new Set<string>(templates.map((t) => t.id));

  while (end <= value.length - 1) {
    if (
      value.slice(end, end + 2) === TEMPLATE_BRACKET_RIGHT &&
      value.slice(start, start + 2) === TEMPLATE_BRACKET_LEFT
    ) {
      // When the end pointer meet the "}}" and the start pointer is "{{"
      // we can extract the string slice as template or normal string.
      const str = value.slice(start + 2, end);
      if (templateSet.has(str)) {
        res.push({
          value: str,
          type: InputType.Template,
        });
      } else {
        res.push({
          value: `${TEMPLATE_BRACKET_LEFT}${str}${TEMPLATE_BRACKET_RIGHT}`,
          type: InputType.String,
        });
      }
      end += 2;
      start = end;
    } else if (value.slice(end, end + 2) === TEMPLATE_BRACKET_LEFT) {
      // When the end pointer meet the "{{"
      // we should reset the position of the start pointer.
      res.push({
        value: value.slice(start, end),
        type: InputType.String,
      });
      start = end;
      end += 2;
    } else {
      end += 1;
    }
  }

  if (start < end) {
    res.push({
      value: value.slice(start, end),
      type: InputType.String,
    });
  }

  // Join the adjacent string value
  return res.reduce((result, data) => {
    if (data.type === InputType.Template) {
      return [...result, data];
    }

    let str = data.value;

    if (
      result.length > 0 &&
      result[result.length - 1].type === InputType.String
    ) {
      str = `${result.pop()?.value ?? ""}${str}`;
    }

    return [
      ...result,
      {
        value: str,
        type: InputType.String,
      },
    ];
  }, [] as TemplateInput[]);
};

// templateInputsToString will convert TemplateInput array into string
// For example:
// [{value: "abc", type: "string"}, {value: "template", type: "template"}] -> "abc{{template}}"
export const templateInputsToString = (inputs: TemplateInput[]): string => {
  return inputs
    .map((input) =>
      input.type === InputType.String
        ? input.value
        : `${TEMPLATE_BRACKET_LEFT}${input.value}${TEMPLATE_BRACKET_RIGHT}`
    )
    .join("");
};
