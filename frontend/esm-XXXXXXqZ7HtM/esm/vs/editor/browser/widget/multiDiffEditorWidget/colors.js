/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { localizeWithPath } from '../../../../nls.js';
import { registerColor } from '../../../../platform/theme/common/colorRegistry.js';
export const multiDiffEditorHeaderBackground = registerColor('multiDiffEditor.headerBackground', { dark: '#808080', light: '#b4b4b4', hcDark: '#808080', hcLight: '#b4b4b4', }, localizeWithPath('vs/editor/browser/widget/multiDiffEditorWidget/colors', 'multiDiffEditor.headerBackground', 'The background color of the diff editor\'s header'));
