// ============================================================================
// MODULE: Messages UI Manager (Con reposicionamiento automático)
// ============================================================================

class MessagesUI {
    constructor() {
        this.container = null;
        this.messagesData = [];
    }
    
    init(containerElement) {
        this.container = containerElement;
    }
    
    render(messages = []) {
        this.messagesData = messages.length > 0 ? messages : [this.getEmptyMessage()];
        
        this.container.innerHTML = `
            <div class="messages-header">
                <h4>Message Configuration</h4>
                <button type="button" class="btn-add-message" onclick="messagesUI.addMessage()">
                    ➕ Add Message
                </button>
            </div>
            <div class="messages-list">
                ${this.messagesData.map((msg, index) => this.renderMessageCard(msg, index)).join('')}
            </div>
        `;
        
        // IMPORTANTE: Reposicionar el popup después de renderizar
        this.repositionPopup();
    }
    
    /**
     * Reposicionar el popup padre después de cambios en el contenido
     */
    repositionPopup() {
        requestAnimationFrame(() => {
            if (popupTracker && popupTracker.activePopup) {
                popupTracker.updatePosition('center');
            }
        });
    }
    
    renderMessageCard(message, index) {
        const functionCodes = CONSTANTS.FUNCTION_CODES;
        const fc = message.function_code || 3;
        const needsCount = [1, 2, 3, 4].includes(fc);
        const needsValues = [5, 6, 15, 16].includes(fc);
        const needsStartAddress = fc !== 43;
        
        return `
            <div class="message-card" data-index="${index}">
                <div class="message-card-header">
                    <span class="message-number">Message #${index + 1}</span>
                    <button type="button" class="btn-delete-message" 
                            onclick="messagesUI.deleteMessage(${index})" 
                            ${this.messagesData.length <= 1 ? 'disabled' : ''}>
                        🗑️
                    </button>
                </div>
                
                <div class="message-card-body">
                    <div class="form-grid">
                        <div class="form-field">
                            <label>Timestamp (ms)</label>
                            <input type="number" class="input-timestamp" 
                                   value="${message.timestamp || 0}" min="0" placeholder="0">
                        </div>
                        
                        <div class="form-field">
                            <label>Function Code</label>
                            <select class="input-function-code" 
                                    onchange="messagesUI.onFunctionCodeChange(${index})">
                                ${functionCodes.map(code => 
                                    `<option value="${code}" ${code === fc ? 'selected' : ''}>
                                        ${code} - ${this.getFunctionCodeName(code)}
                                    </option>`
                                ).join('')}
                            </select>
                        </div>
                        
                        <div class="form-field form-field-checkbox">
                            <label>
                                <input type="checkbox" class="input-recurrent" 
                                       ${message.recurrent ? 'checked' : ''}
                                       onchange="messagesUI.onRecurrentChange(${index})">
                                <span>Recurrent</span>
                            </label>
                        </div>
                        
                        <div class="form-field ${message.recurrent ? '' : 'hidden'}" 
                             data-field="interval-${index}">
                            <label>Interval (ms)</label>
                            <input type="number" class="input-interval" 
                                   value="${message.interval || 1000}" min="1" placeholder="1000">
                        </div>
                        
                        <div class="form-field ${needsStartAddress ? '' : 'hidden'}" 
                             data-field="start-address-${index}">
                            <label>Start Address</label>
                            <input type="text" class="input-start-address" 
                                   value="${message.start_address || '0x0000'}" placeholder="0x0000">
                        </div>
                        
                        <div class="form-field ${needsCount ? '' : 'hidden'}" 
                             data-field="count-${index}">
                            <label>Count</label>
                            <input type="number" class="input-count" 
                                   value="${message.count || 10}" min="1" placeholder="10">
                        </div>
                        
                        <div class="form-field form-field-full ${needsValues ? '' : 'hidden'}" 
                             data-field="values-${index}">
                            <label>Values (comma-separated)</label>
                            <input type="text" class="input-values" 
                                   value="${Array.isArray(message.values) ? message.values.join(',') : ''}" 
                                   placeholder="0,1,2,3">
                            <small class="field-hint">Example: 0,1,0,1 or 100,200,300</small>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }
    
    getFunctionCodeName(code) {
        const names = {
            1: 'Read Coils', 2: 'Read Discrete Inputs',
            3: 'Read Holding Registers', 4: 'Read Input Registers',
            5: 'Write Single Coil', 6: 'Write Single Register',
            15: 'Write Multiple Coils', 16: 'Write Multiple Registers',
            43: 'Read Device ID'
        };
        return names[code] || 'Unknown';
    }
    
    getEmptyMessage() {
        return {
            timestamp: 0,
            recurrent: false,
            interval: 1000,
            function_code: 3,
            start_address: '0x0000',
            count: 10,
            values: []
        };
    }
    
    addMessage() {
        this.messagesData.push(this.getEmptyMessage());
        this.render(this.messagesData);
    }
    
    deleteMessage(index) {
        if (this.messagesData.length <= 1) {
            UIUtils.showError('At least one message is required');
            return;
        }
        this.messagesData.splice(index, 1);
        this.render(this.messagesData);
    }
    
    onFunctionCodeChange(index) {
        const card = document.querySelector(`.message-card[data-index="${index}"]`);
        if (!card) return;
        
        const functionCode = parseInt(card.querySelector('.input-function-code').value);
        const needsCount = [1, 2, 3, 4].includes(functionCode);
        const needsValues = [5, 6, 15, 16].includes(functionCode);
        const needsStartAddress = functionCode !== 43;
        
        const countField = document.querySelector(`[data-field="count-${index}"]`);
        const valuesField = document.querySelector(`[data-field="values-${index}"]`);
        const startAddressField = document.querySelector(`[data-field="start-address-${index}"]`);
        
        if (countField) countField.classList.toggle('hidden', !needsCount);
        if (valuesField) valuesField.classList.toggle('hidden', !needsValues);
        if (startAddressField) startAddressField.classList.toggle('hidden', !needsStartAddress);
        
        // IMPORTANTE: Reposicionar después de mostrar/ocultar campos
        this.repositionPopup();
    }
    
    onRecurrentChange(index) {
        const card = document.querySelector(`.message-card[data-index="${index}"]`);
        if (!card) return;
        
        const isRecurrent = card.querySelector('.input-recurrent').checked;
        const intervalField = document.querySelector(`[data-field="interval-${index}"]`);
        
        if (intervalField) {
            intervalField.classList.toggle('hidden', !isRecurrent);
        }
        
        // IMPORTANTE: Reposicionar después de mostrar/ocultar campos
        this.repositionPopup();
    }
    
    parseMessages() {
        const cards = document.querySelectorAll('.message-card');
        const messages = [];
        const invalidIndexes = [];
        
        cards.forEach((card, index) => {
            const message = {
                timestamp: parseInt(card.querySelector('.input-timestamp').value) || 0,
                recurrent: card.querySelector('.input-recurrent').checked,
                interval: parseInt(card.querySelector('.input-interval').value) || 1000,
                function_code: parseInt(card.querySelector('.input-function-code').value),
                start_address: card.querySelector('.input-start-address').value || '0x0000',
                count: parseInt(card.querySelector('.input-count').value) || 0,
                values: this.parseValues(card.querySelector('.input-values').value)
            };
            
            if (Validators.validateMessageFields(message)) {
                messages.push(message);
            } else {
                invalidIndexes.push(index + 1);
            }
        });
        
        if (invalidIndexes.length > 0) {
            UIUtils.showError(`Invalid fields in message(s): ${invalidIndexes.join(', ')}`);
        }
        
        return messages;
    }
    
    parseValues(valuesStr) {
        if (!valuesStr || valuesStr.trim() === '') return [];
        try {
            return JSON.parse(`[${valuesStr}]`);
        } catch (e) {
            return [];
        }
    }
}

const messagesUI = new MessagesUI();
