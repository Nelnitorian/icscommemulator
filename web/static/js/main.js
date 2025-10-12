// ============================================================================
// MODULE: Main Application & Initialization
// ============================================================================

// Initialize Cytoscape
const cy = cytoscape({
    container: document.getElementById('cy'),
    elements: {
        nodes: networkData.nodes,
        edges: networkData.edges
    },
    style: CYTOSCAPE_STYLE,
    layout: {
        name: 'preset'
    }
});

// Register Cytoscape event handlers
cy.on('taphold', evt => EventHandlers.handleTaphold(evt));
cy.on('tap', evt => EventHandlers.handleTap(evt));

// Register keyboard event handlers
document.addEventListener('keydown', evt => EventHandlers.handleKeyDown(evt));

// DOM ready event handlers
document.addEventListener('DOMContentLoaded', () => {
    // Role change handler
    DOM.fields.role.addEventListener('change', () => EventHandlers.handleRoleChange());

    // Register type change handlers
    DOM.registers.discreteInputsType.addEventListener('change', function() {
        EventHandlers.updateRegisterPlaceholder(this, DOM.registers.discreteInputs);
    });
    DOM.registers.coilsType.addEventListener('change', function() {
        EventHandlers.updateRegisterPlaceholder(this, DOM.registers.coils);
    });
    DOM.registers.inputRegistersType.addEventListener('change', function() {
        EventHandlers.updateRegisterPlaceholder(this, DOM.registers.inputRegisters);
    });
    DOM.registers.holdingRegistersType.addEventListener('change', function() {
        EventHandlers.updateRegisterPlaceholder(this, DOM.registers.holdingRegisters);
    });
});

// Auto-layout if all nodes at origin
if (areAllNodesAtOrigin()) {
    cy.layout({
        name: 'concentric',
        concentric: node => node.degree(),
        levelWidth: () => 1,
        spacingFactor: 2,
        padding: 100
    }).run();
}

// ============================================================================
// Utility functions
// ============================================================================
function areAllNodesAtOrigin() {
    return cy.nodes().every(node => {
        const pos = node.position();
        return pos.x === 0 && pos.y === 0;
    });
}

function zoomIn() {
    cy.zoom(cy.zoom() * 1.2);
    cy.center();
}

function zoomOut() {
    cy.zoom(cy.zoom() * 0.8);
    cy.center();
}

function showRegistersConfiguration(event) {
    EventHandlers.toggleRegistersPanel(event);
}

function showIdentityConfiguration(event) {
    EventHandlers.toggleIdentityPanel(event);
}

// Global function for adding edge rows
function addRow() {
    EdgeManager.addMessageRow();
}

// ============================================================================
// Save Network
// ============================================================================
function save() {
    // VALIDAR ANTES DE GUARDAR
    const validation = ClientValidator.validateScenario();
    if (!ClientValidator.showValidationResults(validation.errors, validation.warnings)) {
        return;
    }

    const networkJson = cy.json();
    const apiData = APITransformer.cytoscapeToAPI(networkJson);

    fetch(`/api/networks/${State.scenarioId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(apiData)
    })
    .then(response => response.json())
    .then(result => {
        if (result.status === 200 || result.message) {
            UIUtils.showError('Saved successfully!', CONSTANTS.LOG_LEVEL.WARNING);
        } else {
            throw new Error(result.error || 'Unknown error');
        }
    })
    .catch(error => {
        UIUtils.showError(`Error: ${error.message}`);
    });
}

// ============================================================================
// Run Simulation
// ============================================================================
function run() {
    DOM.run.settings.style.display = 'block';
    DOM.run.simulationTime.value = '100'; // 100 segundos
    UIUtils.setElementCenter(DOM.run.settings);
}

function cancelRun() {
    DOM.run.settings.style.display = 'none';
    popupTracker.detach();
}

function executeRun() {
    const simulationTime = parseInt(DOM.run.simulationTime.value);
    if (!simulationTime || simulationTime <= 0) {
        UIUtils.showError("Invalid simulation time");
        return;
    }

    // Validar escenario antes de ejecutar
    const validation = ClientValidator.validateScenario();
    if (!ClientValidator.showValidationResults(validation.errors, validation.warnings)) {
        return;
    }

    // Cerrar el popup de settings antes de mostrar el overlay
    DOM.run.settings.style.display = 'none';
    popupTracker.detach();
    
    // Mostrar el overlay de progreso
    DOM.run.overlay.style.display = 'flex';
    DOM.run.timeProgress.textContent = '0';
    DOM.run.percentageProgress.textContent = '0';
    DOM.run.pcapSize.textContent = '0';

    const networkJson = cy.json();
    const apiData = APITransformer.cytoscapeToAPI(networkJson);
    apiData.simulation_time = simulationTime;

    console.log('Running simulation', apiData);

    fetch('/api/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(apiData)
    })
    .then(response => {
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        return response.json();
    })
    .then(result => {
        console.log('Run result:', result);
        
        // El backend devuelve: { message: "Scenario running", file_path: "...", simulation_time: 10 }
        if (result.message === "Scenario running" || result.file_path) {
            State.filePath = result.file_path;
            pollProgress(); // Comenzar polling
        } else if (result.error) {
            throw new Error(result.error);
        } else {
            throw new Error('Unknown error');
        }
    })
    .catch(error => {
        console.error('Error:', error);
        UIUtils.showError('Error: ' + error.message);
        DOM.run.overlay.style.display = 'none';
    });
}

function pollProgress() {
    State.intervalId = setInterval(() => {
        fetch('/api/run', {
            method: 'GET'
        })
        .then(response => {
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return response.json();
        })
        .then(result => {
            console.log('Progress result:', result);
            
            // FIX CRÍTICO: El backend devuelve directamente:
            // { elapsed_seconds: 1, total_seconds: 10, pcap_size: 0, running: true }
            // NO tiene result.status ni result.progress
            
            if (result.elapsed_seconds !== undefined && result.total_seconds !== undefined) {
                // Calcular porcentaje
                const percentage = (result.elapsed_seconds / result.total_seconds) * 100;
                
                // Actualizar UI
                DOM.run.timeProgress.textContent = result.elapsed_seconds;
                DOM.run.percentageProgress.textContent = Math.floor(percentage);
                DOM.run.pcapSize.textContent = result.pcap_size || 0;

                // Detectar cuando termina
                if (!result.running || result.elapsed_seconds >= result.total_seconds) {
                    clearInterval(State.intervalId);
                    setTimeout(() => {
                        alert(`Simulation completed! PCAP saved to: ${State.filePath}`);
                        DOM.run.overlay.style.display = 'none';
                    }, 1000);
                }
            }
        })
        .catch(error => {
            console.error('Error polling progress:', error);
            clearInterval(State.intervalId);
            DOM.run.overlay.style.display = 'none';
        });
    }, 1000);
}

function cancelSimulation() {
    if (State.intervalId) {
        clearInterval(State.intervalId);
    }
    
    // DELETE a /api/run para cancelar
    fetch('/api/run', {
        method: 'DELETE'
    })
    .then(response => response.json())
    .then(result => {
        console.log('Simulation cancelled successfully');
    })
    .catch(error => {
        console.error('Error cancelling simulation:', error);
    })
    .finally(() => {
        DOM.run.overlay.style.display = 'none';
    });
}
