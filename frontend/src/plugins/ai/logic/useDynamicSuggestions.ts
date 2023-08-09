import { Axios, type AxiosResponse } from "axios";
import { head, uniq, values } from "lodash-es";
import { computed, reactive, ref } from "vue";
import { hashCode } from "@/bbkit/BBUtil";
import { WebStorageHelper } from "@/utils";
import { OpenAIMessage, OpenAIResponse } from "../types";
import { useAIContext } from "./context";
import { databaseMetadataToText } from "./utils";

export type SuggestionContext = {
  schema: string; // schema text
  key: string; // a hash key used by storage
  suggestions: string[];
  ready: boolean;
  state: "LOADING" | "IDLE" | "ENDED";
  used: Set<string>;
  consume: (sug: string) => void;
  fetch: () => Promise<string[]>; // returns empty means reaches the end
};

const cache = ref(new Map<string, SuggestionContext>());
const storage = new WebStorageHelper("bb.plugin.open-ai.suggestions");
const MAX_STORED_SUGGESTIONS = 10;

const keyOf = (schema: string) => String(hashCode(schema));

export const useDynamicSuggestions = () => {
  const context = useAIContext();

  const text = computed(() => {
    const meta = context.databaseMetadata.value;
    const engine = context.engine.value;

    if (meta && engine) {
      return databaseMetadataToText(meta, engine);
    }
    return "";
  });

  const requestAI = async (messages: OpenAIMessage[]) => {
    const body = {
      model: "gpt-3.5-turbo",
      messages,
      temperature: 0,
      stop: ["#", ";"],
      top_p: 1.0,
      frequency_penalty: 0.0,
      presence_penalty: 0.0,
    };
    const axios = new Axios({
      timeout: 20 * 1000,
      responseType: "json",
    });
    const headers = {
      "Content-Type": "application/json",
      Authorization: `Bearer ${context.openAIKey.value}`,
    };
    const url =
      context.openAIEndpoint.value === ""
        ? "https://api.openai.com/v1/chat/completions"
        : context.openAIEndpoint.value + "/v1/chat/completions";
    try {
      const response: AxiosResponse<string> = await axios.post(
        url,
        JSON.stringify(body),
        {
          headers,
        }
      );

      const data = JSON.parse(response.data) as OpenAIResponse;
      if (data?.error) {
        throw new Error(data.error.message);
      }

      const text = head(data?.choices)?.message.content?.trim() ?? "";
      const card = JSON.parse(text) as Record<string, string>;
      return values(card ?? {});
    } catch (err) {
      return [];
    }
  };

  const createSuggestion = (schema: string) => {
    const suggestion: SuggestionContext = reactive({
      schema,
      key: keyOf(schema),
      suggestions: [],
      state: "IDLE",
      ready: false,
      used: new Set(),
      consume(sug) {
        const { suggestions, used } = suggestion;
        const index = suggestions.indexOf(sug);
        if (index >= 0) {
          suggestions.splice(index, 1);
        }
        used.add(sug);
        if (suggestions.length === 0) {
          suggestion.fetch();
        }
      },
      async fetch() {
        const { used, suggestions, key } = suggestion;
        const commands = [
          `You are an assistant who works as a Magic: The Suggestion card designer. Create cards that are in the following card schema and JSON format. OUTPUT MUST FOLLOW THIS CARD SCHEMA AND JSON FORMAT. DO NOT EXPLAIN THE CARD.`,
          `{"suggestion-1": "What is the average salary of employees in each department?", "suggestion-2": "What is the average salary of employees in each department?", "suggestion-3": "What is the average salary of employees in each department?"}`,
        ];
        const prompts = [
          schema,
          "Create a suggestion card about interesting queries to try in this database.",
        ];
        if (used.size > 0 || suggestions.length > 0) {
          prompts.push("queries below should be ignored");
          for (const sug of used.values()) {
            prompts.push(sug);
          }
          for (const sug of suggestions) {
            prompts.push(sug);
          }
        }
        const messages: OpenAIMessage[] = [
          {
            role: "system",
            content: commands.join("\n"),
          },
          {
            role: "user",
            content: prompts.join("\n"),
          },
        ];

        suggestion.state = "LOADING";
        const more = (await requestAI(messages)).filter((sug) => {
          if (used.has(sug)) return false;
          if (suggestions.includes(sug)) return false;
          return true;
        });
        suggestions.push(...more);
        suggestion.state = more.length === 0 ? "ENDED" : "IDLE";

        const combined = uniq([...suggestions, ...used.values()]).slice(
          0,
          MAX_STORED_SUGGESTIONS
        );
        if (combined.length > 0) {
          storage.save(key, combined);
        }

        return more;
      },
    });
    const stored = storage.load<string[]>(suggestion.key, []);
    if (stored && stored.length > 0) {
      suggestion.ready = true;
      suggestion.suggestions = stored;
    } else {
      suggestion.fetch().then(() => {
        suggestion.ready = true;
      });
    }
    cache.value.set(schema, suggestion);
    return suggestion;
  };

  const getOrCreateSuggestion = (schema: string) => {
    const cached = cache.value.get(schema);
    if (cached) return cached;
    return createSuggestion(schema);
  };

  return computed(() => {
    if (!text.value) return undefined;
    return getOrCreateSuggestion(text.value);
  });
};
