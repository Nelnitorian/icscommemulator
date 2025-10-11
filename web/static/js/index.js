// ===================================
// CONSTANTS
// ===================================
const API_ENDPOINTS = {
    NETWORKS: '/api/networks/'
};

const MESSAGES = {
    INVALID_IP: 'Please enter a valid IP subrange (e.g., 192.168.100.0/24).',
    INVALID_MASTER_NODES: 'Please enter a valid number of master nodes.',
    INVALID_SLAVE_NODES: 'Please enter a valid number of slave nodes.',
    ERROR_PREFIX: 'Error creating the scenario: '
};

// ===================================
// DOM ELEMENTS CACHE
// ===================================
const elements = {
    formContainer: () => document.getElementById('form-container'),
    loadContainer: () => document.getElementById('load-container'),
    scenarioForm: () => document.getElementById('scenario-form'),
    scenarioList: () => document.getElementById('scenario-list'),
    newScenarioBtn: () => document.getElementById('newScenarioButton'),
    loadScenarioBtn: () => document.getElementById('loadScenarioButton'),
    projectName: () => document.getElementById('projectName'),
    ipSubrange: () => document.getElementById('ipSubrange'),
    protocol: () => document.getElementById('protocol'),
    masterNodes: () => document.getElementById('masterNodes'),
    slaveNodes: () => document.getElementById('slaveNodes')
};

// ===================================
// UI STATE MANAGEMENT
// ===================================
const UIState = {
    clearActiveStates() {
        elements.formContainer().style.display = 'none';
        elements.loadContainer().style.display = 'none';
        elements.newScenarioBtn().classList.remove('active');
        elements.loadScenarioBtn().classList.remove('active');
    },
    
    showForm() {
        this.clearActiveStates();
        elements.formContainer().style.display = 'block';
        elements.newScenarioBtn().classList.add('active');
    },
    
    showLoadContainer() {
        this.clearActiveStates();
        elements.loadContainer().style.display = 'block';
        elements.loadScenarioBtn().classList.add('active');
    }
};

// ===================================
// IP ADDRESS UTILITIES
// ===================================
const IPUtils = {
    parseNetwork(subnet) {
        try {
            const mask = subnet.split('/')[1];
            const networkAddr = ipaddr.IPv4.networkAddressFromCIDR(subnet);
            return `${networkAddr.toString()}/${mask}`;
        } catch (error) {
            return null;
        }
    },
    
    getNextIP(ipAddress, subnet) {
        const ip = ipaddr.parse(ipAddress);
        const subnetParsed = ipaddr.parseCIDR(subnet);
        
        if (!ip.match(subnetParsed)) {
            throw new Error('The IP does not belong to the specified subnet.');
        }
        
        const nextIp = ip.toByteArray();
        
        for (let i = nextIp.length - 1; i >= 0; i--) {
            if (nextIp[i] < 255) {
                nextIp[i]++;
                break;
            }
            nextIp[i] = 0;
        }
        
        return ipaddr.fromByteArray(nextIp).toString();
    }
};

// ===================================
// FORM VALIDATION
// ===================================
const FormValidator = {
    validateIPSubrange(ipSubrange) {
        const validIP = IPUtils.parseNetwork(ipSubrange);
        if (!validIP) {
            alert(MESSAGES.INVALID_IP);
            return null;
        }
        return validIP;
    },
    
    validateNodeCount(value, fieldName) {
        const count = parseInt(value);
        if (isNaN(count) || count <= 0) {
            alert(fieldName === 'master' ? MESSAGES.INVALID_MASTER_NODES : MESSAGES.INVALID_SLAVE_NODES);
            return null;
        }
        return count;
    },
    
    sanitizeProjectName(name) {
        return name.replace(/\s/g, '_');
    }
};

// ===================================
// API COMMUNICATION
// ===================================
const API = {
    async createNetwork(data) {
        const response = await fetch(API_ENDPOINTS.NETWORKS, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        return response.json();
    },
    
    async fetchScenarios() {
        const response = await fetch(API_ENDPOINTS.NETWORKS);
        return response.json();
    }
};

// ===================================
// SCENARIO MANAGEMENT
// ===================================
const ScenarioManager = {
    async loadList() {
        try {
            const scenarios = await API.fetchScenarios();
            this.renderScenarioList(scenarios);
        } catch (error) {
            console.error('Error fetching scenarios:', error);
            alert('Failed to load scenarios. Please try again.');
        }
    },
    
    renderScenarioList(scenarios) {
        const listContainer = elements.scenarioList();
        listContainer.innerHTML = '';
        
        scenarios.forEach(scenario => {
            const item = document.createElement('div');
            item.className = 'scenario-item';
            item.textContent = scenario;
            item.addEventListener('click', () => this.loadScenario(scenario));
            listContainer.appendChild(item);
        });
    },
    
    loadScenario(scenarioName) {
        window.location.href = `networks/${scenarioName}`;
    },
    
    async create(formData) {
        try {
            const result = await API.createNetwork(formData);
            
            if (result.status === 200) {
                console.log('Network created successfully:', result);
                window.location.href = `networks/${formData.projectName}`;
            } else {
                throw new Error(result.error || 'Unknown error');
            }
        } catch (error) {
            console.error('Error creating scenario:', error);
            alert(MESSAGES.ERROR_PREFIX + error.message);
        }
    }
};

// ===================================
// PUBLIC API / EVENT HANDLERS
// ===================================
function loadScenario() {
    UIState.showLoadContainer();
    ScenarioManager.loadList();
}

function showForm() {
    UIState.showForm();
}

function submitForm(event) {
    event.preventDefault();
    
    // Get form values
    const projectName = elements.projectName().value;
    const ipSubrange = elements.ipSubrange().value;
    const protocol = elements.protocol().value;
    const masterNodes = elements.masterNodes().value;
    const slaveNodes = elements.slaveNodes().value;
    
    // Validate inputs
    const validIP = FormValidator.validateIPSubrange(ipSubrange);
    if (!validIP) return;
    
    const validMasterCount = FormValidator.validateNodeCount(masterNodes, 'master');
    if (validMasterCount === null) return;
    
    const validSlaveCount = FormValidator.validateNodeCount(slaveNodes, 'slave');
    if (validSlaveCount === null) return;
    
    // Prepare data
    const parsedProjectName = FormValidator.sanitizeProjectName(projectName);
    const data = {
        projectName: parsedProjectName,
        ipSubrange: validIP,
        protocol: protocol,
        masterNodes: validMasterCount,
        slaveNodes: validSlaveCount
    };
    
    // Submit
    ScenarioManager.create(data);
}

function cancelForm() {
    elements.scenarioForm().reset();
    UIState.clearActiveStates();
}
