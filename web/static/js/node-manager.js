// ============================================================================
// MODULE: Node Manager
// ============================================================================

class NodeManager {
    static createNode(evt) {
        const nodeId = State.incrementNodeCount();
        const name = `node${nodeId}`;
        
        const newNode = cy.add([{
            group: 'nodes',
            classes: CONSTANTS.ROLES.SLAVE,
            data: {
                id: name,
                name: name,
                ...DEFAULT_NODE_TEMPLATE,
                ip: IPManager.assignIP('', ''),
                mac: IPManager.assignIP('', '')
            },
            position: {
                x: evt.position.x,
                y: evt.position.y
            }
        }]);
        
        State.addToHistory(CONSTANTS.ACTIONS.ADD, newNode.json());
    }
    
    static populateForm(data) {
        DOM.fields.name.value = data.name;
        DOM.fields.comment.value = data.comment || '';
        DOM.fields.role.value = data.role || CONSTANTS.ROLES.SLAVE;
        DOM.fields.ip.value = data.ip || '';
        DOM.fields.mac.value = data.mac || '';
        DOM.fields.port.value = data.port || '502';
        DOM.fields.slaveId.value = data.slave_id || '1';
    }
    
    static populateRegisters(data) {
        const regs = DOM.registers;
        
        regs.discreteInputsType.value = data.discrete_inputs.type || CONSTANTS.REGISTER_TYPES.SEQUENTIAL;
        regs.discreteInputs.value = UIUtils.registerToString(
            data.discrete_inputs.values, 
            regs.discreteInputsType.value
        );
        
        regs.coilsType.value = data.coils.type || CONSTANTS.REGISTER_TYPES.SEQUENTIAL;
        regs.coils.value = UIUtils.registerToString(
            data.coils.values,
            regs.coilsType.value
        );
        
        regs.inputRegistersType.value = data.input_registers.type || CONSTANTS.REGISTER_TYPES.SEQUENTIAL;
        regs.inputRegisters.value = UIUtils.registerToString(
            data.input_registers.values,
            regs.inputRegistersType.value
        );
        
        regs.holdingRegistersType.value = data.holding_registers.type || CONSTANTS.REGISTER_TYPES.SEQUENTIAL;
        regs.holdingRegisters.value = UIUtils.registerToString(
            data.holding_registers.values,
            regs.holdingRegistersType.value
        );
    }
    
    static populateIdentity(data) {
        const identity = data.identity || {};
        const fields = DOM.identity;
        
        fields.vendorName.value = identity.vendor_name || '';
        fields.productCode.value = identity.product_code || '';
        fields.majorMinorRevision.value = identity.major_minor_revision || '';
        fields.vendorUrl.value = identity.vendor_url || '';
        fields.productName.value = identity.product_name || '';
        fields.modelName.value = identity.model_name || '';
        fields.userApplicationName.value = identity.user_application_name || '';
    }
    
    static saveConfig() {
        const data = State.selectedElement.data();
        
        data.name = DOM.fields.name.value || data.name;
        data.comment = DOM.fields.comment.value;
        data.role = DOM.fields.role.value;
        data.ip = IPManager.assignIP(DOM.fields.ip.value, data.ip);
        data.mac = Validators.parseMACToColonFormat(DOM.fields.mac.value, data.mac);
        
        if (data.role === CONSTANTS.ROLES.SLAVE) {
            data.port = DOM.fields.port.value;
            data.slave_id = DOM.fields.slaveId.value;
            
            if (DOM.registersPanelElement.style.display === 'block') {
                this.saveRegisters(data);
            } else if (DOM.identityPanelElement.style.display === 'block') {
                this.saveIdentity(data);
            }
        } else if (data.role === CONSTANTS.ROLES.MASTER) {
            // Update edges to slave nodes
            State.selectedElement.connectedEdges().forEach(edge => {
                if (edge.target().data('role') === CONSTANTS.ROLES.SLAVE) {
                    edge.style({ 'target-arrow-shape': CONSTANTS.EDGE.ARROW_SHAPE });
                }
            });
        }
    }
    
    static saveRegisters(data) {
        const regs = DOM.registers;
        
        [data.discrete_inputs.type, data.discrete_inputs.values] = 
            this.parseRegisterField(regs.discreteInputsType.value, regs.discreteInputs.value, 
                data.discrete_inputs.values, data.discrete_inputs.type);
        
        [data.coils.type, data.coils.values] = 
            this.parseRegisterField(regs.coilsType.value, regs.coils.value,
                data.coils.values, data.coils.type);
        
        [data.input_registers.type, data.input_registers.values] = 
            this.parseRegisterField(regs.inputRegistersType.value, regs.inputRegisters.value,
                data.input_registers.values, data.input_registers.type);
        
        [data.holding_registers.type, data.holding_registers.values] = 
            this.parseRegisterField(regs.holdingRegistersType.value, regs.holdingRegisters.value,
                data.holding_registers.values, data.holding_registers.type);
    }
    
    static parseRegisterField(type, value, savedValue, savedType) {
        const parsedValue = Validators.parseRegisterValues(type, value);
        let resultType = type;
        let resultValue = parsedValue;
        
        if (parsedValue === -1) {
            resultValue = "";
        } else if (parsedValue === null) {
            resultValue = savedValue;
            resultType = savedType;
        }
        
        return [resultType, resultValue];
    }
    
    static saveIdentity(data) {
        const fields = DOM.identity;
        
        data.identity = {
            vendor_name: fields.vendorName.value,
            product_code: fields.productCode.value,
            major_minor_revision: fields.majorMinorRevision.value,
            vendor_url: fields.vendorUrl.value,
            product_name: fields.productName.value,
            model_name: fields.modelName.value,
            user_application_name: fields.userApplicationName.value
        };
    }
    
    static showForm(node) {
        this.populateForm(node.data());
        
        DOM.fields.slaveConfig.style.display = 
            DOM.fields.role.value === CONSTANTS.ROLES.MASTER ? 'none' : 'block';
        
        DOM.showNodeConfig();
        
        // Usar el nuevo sistema de tracking
        popupTracker.attachToElement(DOM.nodeConfigPopup, node, 'smart');
    }
    
    static updateFormPosition(node) {
        if (DOM.registersPanelElement.style.display === 'block') {
            popupTracker.positionPanel(DOM.registersPanelElement, DOM.nodeConfigPopup);
        } else if (DOM.identityPanelElement.style.display === 'block') {
            popupTracker.positionPanel(DOM.identityPanelElement, DOM.nodeConfigPopup);
        }
    }
}
