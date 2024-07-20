/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { AsyncIterableObject } from '../../../../base/common/async.js';
import { CancellationToken } from '../../../../base/common/cancellation.js';
import { onUnexpectedExternalError } from '../../../../base/common/errors.js';
import { registerModelAndPositionCommand } from '../../../browser/editorExtensions.js';
import { ILanguageFeaturesService } from '../../../common/services/languageFeatures.js';
export class HoverProviderResult {
    constructor(provider, hover, ordinal) {
        this.provider = provider;
        this.hover = hover;
        this.ordinal = ordinal;
    }
}
async function executeProvider(provider, ordinal, model, position, token) {
    try {
        const result = await Promise.resolve(provider.provideHover(model, position, token));
        if (result && isValid(result)) {
            return new HoverProviderResult(provider, result, ordinal);
        }
    }
    catch (err) {
        onUnexpectedExternalError(err);
    }
    return undefined;
}
export function getHover(registry, model, position, token) {
    const providers = registry.ordered(model);
    const promises = providers.map((provider, index) => executeProvider(provider, index, model, position, token));
    return AsyncIterableObject.fromPromises(promises).coalesce();
}
export function getHoverPromise(registry, model, position, token) {
    return getHover(registry, model, position, token).map(item => item.hover).toPromise();
}
registerModelAndPositionCommand('_executeHoverProvider', (accessor, model, position) => {
    const languageFeaturesService = accessor.get(ILanguageFeaturesService);
    return getHoverPromise(languageFeaturesService.hoverProvider, model, position, CancellationToken.None);
});
function isValid(result) {
    const hasRange = (typeof result.range !== 'undefined');
    const hasHtmlContent = typeof result.contents !== 'undefined' && result.contents && result.contents.length > 0;
    return hasRange && hasHtmlContent;
}
