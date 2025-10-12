// ============================================================================
// MODULE: Event Handlers (DEBUG VERSION)
// ============================================================================

class EventHandlers {
    static handleTaphold(evt) {

        State.setLastTaphold();
        const element = evt.target;

        if (element === cy) {
            NodeManager.createNode(evt);
        } else if (element.isNode()) {
            State.selectElement(element);
            NodeManager.showForm(element);
        } else if (element.isEdge()) {
            State.selectElement(element);
            EdgeManager.showForm(element);
        }
    }

    static handleTap(evt) {
        // Ignore taps immediately after taphold
        if (State.isTapholdRecent()) {
            return;
        }

        // Save any open configurations
        this.saveOpenConfigurations();

        const element = evt.target;

        if (element === cy) {
            if (State.selectedElement) {
                State.unselectElement();
            }
        } else if (element.isNode()) {
            this.handleNodeTap(element);
        } else if (element.isEdge()) {
            this.handleEdgeTap(element);
        }
    }

    static handleNodeTap(node) {
        if (!State.selectedElement) {
            State.selectElement(node);
            return;
        }

        if (State.selectedElement.isNode()) {
            if (State.selectedElement === node) {
                State.unselectElement();
            } else {
                EdgeManager.createEdge(State.selectedElement, node);
                State.unselectElement();
            }
        } else {
            State.selectElement(node);
        }
    }

    static handleEdgeTap(edge) {

        if (!State.selectedElement) {
            State.selectElement(edge);
        } else if (State.selectedElement !== edge) {
            State.unselectElement();
            State.selectElement(edge);
        } else {
            State.unselectElement();
        }
    }

    static handleKeyDown(evt) {
        const key = evt.key;

        // Ignore if popups are visible
        if (DOM.isAnyPopupVisible()) {
            if (key === CONSTANTS.KEYS.ESCAPE) {
                this.saveOpenConfigurations();
                State.unselectElement();
            }
            return;
        }

        if ((key === CONSTANTS.KEYS.DELETE || key === CONSTANTS.KEYS.SUPR) && State.selectedElement) {
            this.deleteSelectedElement();
        } else if (key === CONSTANTS.KEYS.ESCAPE) {
            this.saveOpenConfigurations();
            State.unselectElement();
        } else if (evt.ctrlKey) {
            if (key === CONSTANTS.KEYS.Z) {
                State.undo();
            } else if (key === CONSTANTS.KEYS.Y) {
                State.redo();
            }
        }
    }

    static deleteSelectedElement() {
        const connectedEdges = State.selectedElement.connectedEdges().jsons();
        State.addToHistory(CONSTANTS.ACTIONS.DELETE, State.selectedElement.json(), connectedEdges);
        State.selectedElement.remove();
        State.selectedElement = null;
    }

    static saveOpenConfigurations() {
        if (DOM.nodeConfigPopup.style.display === 'block' && State.selectedElement?.isNode()) {
            NodeManager.saveConfig();
        } else if (DOM.edgeConfigPopup.style.display === 'block' && State.selectedElement?.isEdge()) {
            EdgeManager.saveConfig();
        }

        State.removeCanvasListener();
        DOM.hideAllPopups();
    }

    static handleRoleChange() {
        if (State.selectedElement.connectedEdges().length > 0) {
            UIUtils.showError('Error: Cannot change the role of a connected node');
            DOM.fields.role.value = State.selectedElement.data('role');
            return;
        }

        DOM.fields.slaveConfig.style.display = 
            DOM.fields.role.value === CONSTANTS.ROLES.SLAVE ? 'block' : 'none';
    }

    static updateRegisterPlaceholder(typeElement, inputElement) {
        const placeholder = typeElement.value === CONSTANTS.REGISTER_TYPES.SPARSE 
            ? '1:1,2:0,3:1,4:0' 
            : '0,1,0,0,1';
        inputElement.placeholder = placeholder;
    }

    static toggleRegistersPanel(event) {
        event.preventDefault();

        if (DOM.identityPanelElement.style.display === 'block') {
            NodeManager.saveIdentity(State.selectedElement.data());
            DOM.identityPanelElement.style.display = 'none';
            popupTracker.removeSecondaryPanel(DOM.identityPanelElement);
        }

        if (DOM.registersPanelElement.style.display !== 'block') {
            DOM.registersPanelElement.style.display = 'block';
            NodeManager.populateRegisters(State.selectedElement.data());
            popupTracker.positionPanel(DOM.registersPanelElement, DOM.nodeConfigPopup);
            popupTracker.addSecondaryPanel(DOM.registersPanelElement);
        } else {
            NodeManager.saveRegisters(State.selectedElement.data());
            DOM.registersPanelElement.style.display = 'none';
            popupTracker.removeSecondaryPanel(DOM.registersPanelElement);
        }
    }


    static toggleIdentityPanel(event) {
        event.preventDefault();

        if (DOM.registersPanelElement.style.display === 'block') {
            NodeManager.saveRegisters(State.selectedElement.data());
            DOM.registersPanelElement.style.display = 'none';
            popupTracker.removeSecondaryPanel(DOM.registersPanelElement);
        }

        if (DOM.identityPanelElement.style.display !== 'block') {
            DOM.identityPanelElement.style.display = 'block';
            NodeManager.populateIdentity(State.selectedElement.data());
            popupTracker.positionPanel(DOM.identityPanelElement, DOM.nodeConfigPopup);
            popupTracker.addSecondaryPanel(DOM.identityPanelElement);
        } else {
            NodeManager.saveIdentity(State.selectedElement.data());
            DOM.identityPanelElement.style.display = 'none';
            popupTracker.removeSecondaryPanel(DOM.identityPanelElement);
        }
    }
}


let clipboard = null;

document.addEventListener('keydown', function(evt) {
    // Copy (Ctrl+C)
    if (evt.ctrlKey && evt.key === 'c' && !DOM.isAnyPopupVisible()) {
        if (State.selectedElement) {
            clipboard = {
                type: State.selectedElement.isNode() ? 'node' : 'edge',
                data: JSON.parse(JSON.stringify(State.selectedElement.json()))
            };
        }
    }

    // Paste (Ctrl+V)
    if (evt.ctrlKey && evt.key === 'v' && !DOM.isAnyPopupVisible()) {
        if (clipboard && clipboard.type === 'node') {
            pasteNode();
        }
    }
});

function pasteNode() {
    if (!clipboard || clipboard.type !== 'node') return;

    const nodeId = State.incrementNodeCount();
    const name = `node${nodeId}`;

    const newNodeData = JSON.parse(JSON.stringify(clipboard.data));
    newNodeData.data.id = name;
    newNodeData.data.name = name;

    // Asignar nueva IP
    const existingIPs = cy.nodes().map(node => node.data('ip'));
    newNodeData.data.ip = IPManager.getUniqueIP(existingIPs, networkData.ip_network);

    // Offset de posición
    newNodeData.position.x += 50;
    newNodeData.position.y += 50;

    // CRÍTICO: Limpiar clases de selección
    if (newNodeData.classes) {
        newNodeData.classes = newNodeData.classes
            .replace('selected', '')
            .replace(/\s+/g, ' ')
            .trim();
    }

    const newNode = cy.add(newNodeData);

    // Asegurar que no tiene clase selected
    newNode.removeClass('selected');

    State.addToHistory(CONSTANTS.ACTIONS.ADD, newNode.json());
}
