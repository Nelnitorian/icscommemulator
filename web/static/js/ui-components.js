window.App = window.App || {};

(function () {
    'use strict';
    const CONSTANTS = App.CONSTANTS;
    const t = (key, vars) => (window.I18N ? window.I18N.t(key, vars) : key);

    class UIUtils {
        static showError(message) {
            if (!App.DOM || !App.DOM.errorMessage) {
                alert(message);
                return;
            }
            App.DOM.errorMessage.textContent = message;
            App.DOM.errorMessage.classList.add('show');
            setTimeout(() => App.DOM.errorMessage.classList.remove('show'), 5000);
        }
        static setElementCenter(el) {
            el.style.display = 'block';
            el.style.left = `${(window.innerWidth - el.offsetWidth) / 2}px`;
            el.style.top = `${(window.innerHeight - el.offsetHeight) / 2}px`;
        }
        static registerToString(reg, type) {
            if (!reg) return '';
            if (type === CONSTANTS.REGISTER_TYPES.SEQUENTIAL) {
                return Array.isArray(reg) ? reg.join(',') : '';
            }
            return Object.keys(reg).map(k => `${k}:${reg[k]}`).join(',');
        }
    }

    class DOMManager {
        constructor() {
            this.cacheElements();
        }
        cacheElements() {
            this.nodeConfigPopup = document.getElementById('configPopup');
            this.edgeConfigPopup = document.getElementById('edgeConfigPopup');
            this.registersPanelElement = document.getElementById('registersPanel');
            this.identityPanelElement = document.getElementById('identityPanel');
            this.errorMessage = document.getElementById('error-message');

            this.fields = {
                name: document.getElementById('name'),
                comment: document.getElementById('comment'),
                role: document.getElementById('role'),
                slaveConfig: document.getElementById('slave-config'),
                ip: document.getElementById('ip'),
                mac: document.getElementById('mac'),
                port: document.getElementById('port'),
                slaveId: document.getElementById('slaveid')
            };

            this.iec104Fields = {
                commonAddress: document.getElementById('iec104-common-address'),
                tickRateMs: document.getElementById('iec104-tick-rate'),
                selectTimeoutMs: document.getElementById('iec104-select-timeout'),
                maxConnections: document.getElementById('iec104-max-connections'),
                authorizedMasters: document.getElementById('iec104-authorized-masters'),
                k: document.getElementById('iec104-k'),
                w: document.getElementById('iec104-w'),
                t1: document.getElementById('iec104-t1'),
                t2: document.getElementById('iec104-t2'),
                t3: document.getElementById('iec104-t3')
            };

            this.iec104PointsPanel = document.getElementById('iec104-points-panel');
            this.iec104PointsFields = {
                singlePoints: document.getElementById('iec104-single-points'),
                measuredShort: document.getElementById('iec104-measured-short')
            };

            this.registers = {
                discreteInputsType: document.getElementById('discrete_inputs_type'),
                discreteInputs: document.getElementById('discrete_inputs'),
                coilsType: document.getElementById('coils_type'),
                coils: document.getElementById('coils'),
                inputRegistersType: document.getElementById('input_registers_type'),
                inputRegisters: document.getElementById('input_registers'),
                holdingRegistersType: document.getElementById('holding_registers_type'),
                holdingRegisters: document.getElementById('holding_registers')
            };

            this.identity = {
                vendorName: document.getElementById('vendor_name'),
                productCode: document.getElementById('product_code'),
                majorMinorRevision: document.getElementById('major_minor_revision'),
                vendorUrl: document.getElementById('vendor_url'),
                productName: document.getElementById('product_name'),
                modelName: document.getElementById('model_name'),
                userApplicationName: document.getElementById('user_application_name')
            };

            this.edgeConfigDirection = document.getElementById('edgeConfigDirection');
            this.run = {
                settings: document.getElementById('run_settings'),
                simulationTime: document.getElementById('simulation_time'),
                overlay: document.getElementById('run_overlay'),
                timeProgress: document.getElementById('time_progress'),
                percentageProgress: document.getElementById('percentage_progress'),
                pcapSize: document.getElementById('pcap_size')
            };
        }

        hideAllPopups() {
            if (this.nodeConfigPopup) this.nodeConfigPopup.style.display = 'none';
            if (this.edgeConfigPopup) this.edgeConfigPopup.style.display = 'none';
            if (this.registersPanelElement) this.registersPanelElement.style.display = 'none';
            if (this.identityPanelElement) this.identityPanelElement.style.display = 'none';
            if (this.iec104PointsPanel) this.iec104PointsPanel.style.display = 'none';
        }

        isAnyPopupVisible() {
            return (this.nodeConfigPopup && this.nodeConfigPopup.style.display === 'block') ||
                (this.edgeConfigPopup && this.edgeConfigPopup.style.display === 'block') ||
                (this.run.overlay && this.run.overlay.style.display === 'block') ||
                (this.run.settings && this.run.settings.style.display === 'block');
        }

        showNodeConfig() { this.nodeConfigPopup.style.display = 'block'; }
        showEdgeConfig() { this.edgeConfigPopup.style.display = 'block'; }
    }

    class PopupTracker {
        constructor() {
            this.activePopup = null;
            this.activeElement = null;
            this.secondaryPanels = [];
            this.updateHandler = null;
        }

        attachToElement(popup, element, strategy = 'smart') {
            this.detach();
            this.activePopup = popup;
            this.activeElement = element;
            this.secondaryPanels = [];

            requestAnimationFrame(() => this.updatePosition(strategy));

            if (element.isNode && element.isNode()) {
                this.updateHandler = () => {
                    this.updatePosition(strategy);
                    this.updateSecondaryPanels();
                };
                element.on('position drag', this.updateHandler);
                if (window.cy) window.cy.on('pan zoom viewport', this.updateHandler);
            }
        }

        detach() {
            if (this.updateHandler && this.activeElement && this.activeElement.isNode()) {
                this.activeElement.removeListener('position drag', this.updateHandler);
                if (window.cy) window.cy.off('pan zoom viewport', this.updateHandler);
            }
            this.activePopup = null;
            this.activeElement = null;
        }

        updatePosition(strategy) {
            if (!this.activePopup || !this.activeElement) return;
            if (strategy === 'center' || this.activeElement.isEdge()) {
                this.positionCenter();
            } else {
                const renderedPos = this.activeElement.renderedPosition();
                this.activePopup.style.display = 'block';
                this.activePopup.style.left = `${renderedPos.x + 20}px`;
                this.activePopup.style.top = `${renderedPos.y}px`;
            }
        }

        positionCenter() {
            const popup = this.activePopup;
            popup.style.display = 'block';
            const left = (window.innerWidth - popup.offsetWidth) / 2;
            const top = (window.innerHeight - popup.offsetHeight) / 2;
            popup.style.left = `${left}px`;
            popup.style.top = `${top}px`;
        }

        addSecondaryPanel(panel) { panel.style.display = 'block'; }
        removeSecondaryPanel(panel) { panel.style.display = 'none'; }
        updateSecondaryPanels() { }
        positionPanel(panel, parent) { }
    }

    const popupTracker = new PopupTracker();

    class ProtocolMessagesUI {
        constructor() {
            this.container = null;
            this.messagesData = [];
            this.protocol = null;
        }

        init(container) {
            this.container = container;
            this.protocol = (networkData?.protocol || 'modbus').toLowerCase();
        }

        render(messages = []) {
            this.messagesData = messages.length > 0 ? messages : [this.getEmptyMessage()];
            this.container.innerHTML = `
        <div class="messages-list">${this.messagesData.map((m, i) => this.renderMessageCard(m, i)).join('')}</div>
        <button type="button" class="btn-add-message" onclick="messagesUI.addMessage()">${t('common.add')}</button>
      `;
            if (App.PopupTracker) App.PopupTracker.updatePosition('center');
        }

        renderMessageCard(msg, idx) {
            return `<div class="message-card" data-index="${idx}">
            <div class="message-header">Message ${idx + 1} <button onclick="messagesUI.deleteMessage(${idx})">x</button></div>
            <div class="message-body">
                <label>Function Code <input class="input-function-code" value="${msg.function_code || 3}"></label>
                <label>Values <input class="input-values" value="${Array.isArray(msg.values) ? msg.values.join(',') : ''}"></label>
            </div>
        </div>`;
        }

        getEmptyMessage() { return { function_code: 3, values: [] }; }

        parseMessages() {
            return this.messagesData;
        }

        addMessage() { this.messagesData.push(this.getEmptyMessage()); this.render(this.messagesData); }
        deleteMessage(idx) { this.messagesData.splice(idx, 1); this.render(this.messagesData); }
    }

    const messagesUI = new ProtocolMessagesUI();

    App.DOMManager = DOMManager;
    App.PopupTracker = popupTracker;
    App.UIUtils = UIUtils;
    App.messagesUI = messagesUI;
    window.messagesUI = messagesUI;
}());
