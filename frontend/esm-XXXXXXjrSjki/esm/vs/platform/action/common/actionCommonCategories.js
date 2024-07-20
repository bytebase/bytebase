/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { localizeWithPath } from '../../../nls.js';
export const Categories = Object.freeze({
    View: { value: localizeWithPath('vs/platform/action/common/actionCommonCategories', 'view', "View"), original: 'View' },
    Help: { value: localizeWithPath('vs/platform/action/common/actionCommonCategories', 'help', "Help"), original: 'Help' },
    Test: { value: localizeWithPath('vs/platform/action/common/actionCommonCategories', 'test', "Test"), original: 'Test' },
    File: { value: localizeWithPath('vs/platform/action/common/actionCommonCategories', 'file', "File"), original: 'File' },
    Preferences: { value: localizeWithPath('vs/platform/action/common/actionCommonCategories', 'preferences', "Preferences"), original: 'Preferences' },
    Developer: { value: localizeWithPath('vs/platform/action/common/actionCommonCategories', { key: 'developer', comment: ['A developer on Code itself or someone diagnosing issues in Code'] }, "Developer"), original: 'Developer' }
});
