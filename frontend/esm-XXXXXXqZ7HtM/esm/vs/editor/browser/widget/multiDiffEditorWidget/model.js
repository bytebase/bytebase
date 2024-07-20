/*---------------------------------------------------------------------------------------------
 *  Copyright (c) Microsoft Corporation. All rights reserved.
 *  Licensed under the MIT License. See License.txt in the project root for license information.
 *--------------------------------------------------------------------------------------------*/
import { Event } from '../../../../base/common/event.js';
export class ConstLazyPromise {
    constructor(_value) {
        this._value = _value;
        this.onHasValueDidChange = Event.None;
    }
    request() {
        return Promise.resolve(this._value);
    }
    get value() {
        return this._value;
    }
}
