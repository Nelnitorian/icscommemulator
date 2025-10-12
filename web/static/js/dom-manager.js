// ============================================================================
// MODULE: DOM Elements Manager
// ============================================================================

class DOMManager {
    constructor() {
        this.cacheElements();
    }

    cacheElements() {
        // Popups
        this.nodeConfigPopup = document.getElementById('configPopup');
        this.edgeConfigPopup = document.getElementById('edgeConfigPopup');
        this.registersPanelElement = document.getElementById('registersPanel');
        this.identityPanelElement = document.getElementById('identityPanel');

        // Forms
        this.nodeConfigForm = document.getElementById('configForm');
        this.edgeConfigForm = document.getElementById('edgeConfigForm');

        // Node fields
        this.fields = {
            name: document.getElementById('name'),
            comment: document.getElementById('comment'),
            role: document.getElementById('role'),
            slaveConfig: document.getElementById('slave-config'),
            ip: document.getElementById('ip'),
            mac: document.getElementById('mac'),
            port: document.getElementById('port'),
            slaveId: document.getElementById('slave_id')
        };

        // Register fields
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

        // Identity fields
        this.identity = {
            vendorName: document.getElementById('vendor_name'),
            productCode: document.getElementById('product_code'),
            majorMinorRevision: document.getElementById('major_minor_revision'),
            vendorUrl: document.getElementById('vendor_url'),
            productName: document.getElementById('product_name'),
            modelName: document.getElementById('model_name'),
            userApplicationName: document.getElementById('user_application_name')
        };

        // Edge fields
        this.edgeConfigDirection = document.getElementById('edgeConfigDirection');

        // Run elements - CORREGIDO: IDs con guiones bajos
        this.run = {
            settings: document.getElementById('run_settings'),
            settingsContent: document.getElementById('run_settings_content'),
            simulationTime: document.getElementById('simulation_time'),
            cancelButton: document.getElementById('cancel_button'),
            runButton: document.getElementById('run_button'),
            overlay: document.getElementById('run_overlay'),
            overlayContent: document.getElementById('run_overlay_content'),
            timeProgress: document.getElementById('time_progress'),
            percentageProgress: document.getElementById('percentage_progress'),
            pcapSize: document.getElementById('pcap_size')
        };

        this.errorMessage = document.getElementById('error-message');
    }

    hideAllPopups() {
        this.nodeConfigPopup.style.display = 'none';
        this.edgeConfigPopup.style.display = 'none';
        this.registersPanelElement.style.display = 'none';
        this.identityPanelElement.style.display = 'none';
    }

    isAnyPopupVisible() {
        return this.nodeConfigPopup.style.display === 'block' ||
               this.edgeConfigPopup.style.display === 'block' ||
               this.run.overlay.style.display === 'block' ||
               this.run.settings.style.display === 'block';
    }

    showNodeConfig() {
        this.nodeConfigPopup.style.display = 'block';
    }

    showEdgeConfig() {
        this.edgeConfigPopup.style.display = 'block';
    }
}

const DOM = new DOMManager();
