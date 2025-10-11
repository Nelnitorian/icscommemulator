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
});

// Utility functions
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


function run() {
    DOM.run.settings.style.display = 'block';
    DOM.run.simulationTime.value = '60000';
    UIUtils.setElementCenter(DOM.run.settings);
}

function cancelRun() {
    DOM.run.settings.style.display = 'none';
}

function executeRun() {
    const simulationTime = parseInt(DOM.run.simulationTime.value);
    if (!simulationTime || simulationTime <= 0) {
        UIUtils.showError("Invalid simulation time");
        return;
    }

    DOM.run.settings.style.display = 'none';
    DOM.run.overlay.style.display = 'block';
    DOM.run.timeProgress.textContent = '0';
    DOM.run.percentageProgress.textContent = '0';
    DOM.run.pcapSize.textContent = '0';

    const networkJson = cy.json();
    const apiData = APITransformer.cytoscapeToAPI(networkJson);
    apiData.simulation_time = simulationTime;

    console.log('Running simulation', apiData);

    // CAMBIO 1: Endpoint a /api/run
    fetch('/api/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(apiData)
    })
    .then(response => response.json())
    .then(result => {
        if (result.status === 200) {
            State.filePath = result.filepath;
            pollProgress();  // Comenzar polling
        } else {
            throw new Error(result.error || 'Unknown error');
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
        // CAMBIO 2: GET a /api/run para obtener progreso
        fetch('/api/run', {
            method: 'GET'
        })
        .then(response => response.json())
        .then(result => {
            if (result.status === 200) {
                const progress = result.progress;
                DOM.run.timeProgress.textContent = Math.floor(progress.elapsed_time);
                DOM.run.percentageProgress.textContent = Math.floor(progress.percentage);
                DOM.run.pcapSize.textContent = progress.pcap_size;

                // Cuando termina
                if (progress.percentage >= 100 || progress.completed) {
                    clearInterval(State.intervalId);
                    setTimeout(() => {
                        // CAMBIO 3: Solo mostrar mensaje, NO descargar
                        alert(`Simulation completed! PCAP saved to: ${State.filePath}`);
                        DOM.run.overlay.style.display = 'none';
                    }, 1000);
                }
            }
        })
        .catch(() => {
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
        if (result.status === 200) {
            console.log('Simulation cancelled successfully');
        }
    })
    .catch(error => {
        console.error('Error cancelling simulation:', error);
    })
    .finally(() => {
        DOM.run.overlay.style.display = 'none';
    });
}


function pollProgress() {
    State.intervalId = setInterval(() => {
        fetch(`/api/run`)
            .then(response => response.json())
            .then(result => {
                if (result.status === 200) {
                    const progress = result.progress;
                    
                    DOM.run.timeProgress.textContent = Math.floor(progress.elapsed_time);
                    DOM.run.percentageProgress.textContent = `${Math.floor(progress.percentage)}%`;
                    DOM.run.pcapSize.textContent = progress.pcap_size;
                    
                    if (progress.percentage >= 100 || progress.completed) {
                        clearInterval(State.intervalId);
                        setTimeout(() => {
                            if (confirm('Simulation completed! Download PCAP?')) {
                                window.location.href = `/api/networks/${State.scenarioId}/download`;
                            }
                            DOM.run.overlay.style.display = 'none';
                        }, 1000);
                    }
                }
            })
            .catch(() => {
                clearInterval(State.intervalId);
                DOM.run.overlay.style.display = 'none';
            });
    }, 1000);
}

function cancelSimulation() {
    if (State.intervalId) clearInterval(State.intervalId);
    
    fetch(`/api/networks/${State.scenarioId}/cancel`, { method: 'POST' })
        .finally(() => {
            DOM.run.overlay.style.display = 'none';
        });
}

