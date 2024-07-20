/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
let isPseudo = (typeof document !== 'undefined' && document.location && document.location.hash.indexOf('pseudo=true') >= 0);
const DEFAULT_TAG = 'i-default';
function _format(message, args) {
    let result;
    if (args.length === 0) {
        result = message;
    }
    else {
        result = message.replace(/\{(\d+)\}/g, (match, rest) => {
            const index = rest[0];
            const arg = args[index];
            let result = match;
            if (typeof arg === 'string') {
                result = arg;
            }
            else if (typeof arg === 'number' || typeof arg === 'boolean' || arg === void 0 || arg === null) {
                result = String(arg);
            }
            return result;
        });
    }
    if (isPseudo) {
        // FF3B and FF3D is the Unicode zenkaku representation for [ and ]
        result = '\uFF3B' + result.replace(/[aouei]/g, '$&$&') + '\uFF3D';
    }
    return result;
}
function findLanguageForModule(config, name) {
    let result = config[name];
    if (result) {
        return result;
    }
    result = config['*'];
    if (result) {
        return result;
    }
    return null;
}
function endWithSlash(path) {
    if (path.charAt(path.length - 1) === '/') {
        return path;
    }
    return path + '/';
}
async function getMessagesFromTranslationsService(translationServiceUrl, language, name) {
    const url = endWithSlash(translationServiceUrl) + endWithSlash(language) + 'vscode/' + endWithSlash(name);
    const res = await fetch(url);
    if (res.ok) {
        const messages = await res.json();
        return messages;
    }
    throw new Error(`${res.status} - ${res.statusText}`);
}
function createScopedLocalize(scope) {
    return function (idx, defaultValue) {
        const restArgs = Array.prototype.slice.call(arguments, 2);
        return _format(scope[idx], restArgs);
    };
}
function createScopedLocalize2(scope) {
    return (idx, defaultValue, ...args) => ({
        value: _format(scope[idx], args),
        original: _format(defaultValue, args)
    });
}
/**
 * @skipMangle
 */
export function localize(data, message, ...args) {
    return _format(message, args);
}
let locale = undefined;
let translations = {};
export function setLocale(_locale, _translations) {
    locale = _locale;
    translations = _translations;
}
/**
 * @skipMangle
 */
export function localizeWithPath(path, data, defaultMessage, ...args) {
    const key = typeof data === 'object' ? data.key : data;
    const message = (translations[path] ?? {})[key] ?? defaultMessage;
    return _format(message, args);
}
/**
 * @skipMangle
 */
export function localize2WithPath(path, data, defaultMessage, ...args) {
    const key = typeof data === 'object' ? data.key : data;
    const message = (translations[path] ?? {})[key] ?? defaultMessage;
    return {
        value: _format(message, args),
        original: _format(defaultMessage, args)
    };
}
/**
 * @skipMangle
 */
export function localize2(data, message, ...args) {
    const original = _format(message, args);
    return {
        value: original,
        original
    };
}
/**
 * @skipMangle
 */
export function getConfiguredDefaultLocale(_) {
    // This returns undefined because this implementation isn't used and is overwritten by the loader
    // when loaded.
    return locale;
}
/**
 * @skipMangle
 */
export function setPseudoTranslation(value) {
    isPseudo = value;
}
/**
 * Invoked in a built product at run-time
 * @skipMangle
 */
export function create(key, data) {
    return {
        localize: createScopedLocalize(data[key]),
        localize2: createScopedLocalize2(data[key]),
        getConfiguredDefaultLocale: data.getConfiguredDefaultLocale ?? ((_) => undefined)
    };
}
/**
 * Invoked by the loader at run-time
 * @skipMangle
 */
export function load(name, req, load, config) {
    const pluginConfig = config['vs/nls'] ?? {};
    if (!name || name.length === 0) {
        // TODO: We need to give back the mangled names here
        return load({
            localize: localize,
            localize2: localize2,
            getConfiguredDefaultLocale: () => pluginConfig.availableLanguages?.['*']
        });
    }
    const language = pluginConfig.availableLanguages ? findLanguageForModule(pluginConfig.availableLanguages, name) : null;
    const useDefaultLanguage = language === null || language === DEFAULT_TAG;
    let suffix = '.nls';
    if (!useDefaultLanguage) {
        suffix = suffix + '.' + language;
    }
    const messagesLoaded = (messages) => {
        if (Array.isArray(messages)) {
            messages.localize = createScopedLocalize(messages);
            messages.localize2 = createScopedLocalize2(messages);
        }
        else {
            messages.localize = createScopedLocalize(messages[name]);
            messages.localize2 = createScopedLocalize2(messages[name]);
        }
        messages.getConfiguredDefaultLocale = () => pluginConfig.availableLanguages?.['*'];
        load(messages);
    };
    if (typeof pluginConfig.loadBundle === 'function') {
        pluginConfig.loadBundle(name, language, (err, messages) => {
            // We have an error. Load the English default strings to not fail
            if (err) {
                req([name + '.nls'], messagesLoaded);
            }
            else {
                messagesLoaded(messages);
            }
        });
    }
    else if (pluginConfig.translationServiceUrl && !useDefaultLanguage) {
        (async () => {
            try {
                const messages = await getMessagesFromTranslationsService(pluginConfig.translationServiceUrl, language, name);
                return messagesLoaded(messages);
            }
            catch (err) {
                // Language is already as generic as it gets, so require default messages
                if (!language.includes('-')) {
                    console.error(err);
                    return req([name + '.nls'], messagesLoaded);
                }
                try {
                    // Since there is a dash, the language configured is a specific sub-language of the same generic language.
                    // Since we were unable to load the specific language, try to load the generic language. Ex. we failed to find a
                    // Swiss German (de-CH), so try to load the generic German (de) messages instead.
                    const genericLanguage = language.split('-')[0];
                    const messages = await getMessagesFromTranslationsService(pluginConfig.translationServiceUrl, genericLanguage, name);
                    // We got some messages, so we configure the configuration to use the generic language for this session.
                    pluginConfig.availableLanguages ?? (pluginConfig.availableLanguages = {});
                    pluginConfig.availableLanguages['*'] = genericLanguage;
                    return messagesLoaded(messages);
                }
                catch (err) {
                    console.error(err);
                    return req([name + '.nls'], messagesLoaded);
                }
            }
        })();
    }
    else {
        req([name + suffix], messagesLoaded, (err) => {
            if (suffix === '.nls') {
                console.error('Failed trying to load default language strings', err);
                return;
            }
            console.error(`Failed to load message bundle for language ${language}. Falling back to the default language:`, err);
            req([name + '.nls'], messagesLoaded);
        });
    }
}
