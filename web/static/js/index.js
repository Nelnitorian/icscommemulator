// index.js

function loadScenario() {
    // clear previous state
    document.getElementById('form-container').style.display = 'none';
    document.getElementById('newScenarioButton').classList.remove('active');

    document.getElementById('load-container').style.display = 'block';
    document.getElementById('loadScenarioButton').classList.add('active');

    fetchScenarios();
}

function showForm() {
    // clear previous state
    document.getElementById('load-container').style.display = 'none';
    document.getElementById('loadScenarioButton').classList.remove('active');

    document.getElementById('form-container').style.display = 'block';
    document.getElementById('newScenarioButton').classList.add('active');
}

function submitForm(event) {
    event.preventDefault();

    const projectName = document.getElementById('projectName').value;
    const ipSubrange = document.getElementById('ipSubrange').value;
    const protocol = document.getElementById('protocol').value;
    const masterNodes = document.getElementById('masterNodes').value;
    const slaveNodes = document.getElementById('slaveNodes').value;

    // Validate IP Subrange
    let validIpSubrange = parseIpNetwork(ipSubrange);
    if (!validIpSubrange) {
        alert("Please enter a valid IP subrange (e.g., 192.168.100.0/24).");
        return;
    }

    // Validate number of master and slave nodes
    if (isNaN(masterNodes) || masterNodes <= 0) {
        alert("Please enter a valid number of master nodes  .");
        return;
    }

    if (isNaN(slaveNodes) || slaveNodes <= 0) {
        alert("Please enter a valid number of slave nodes.");
        return;
    }

    // Replace ' ' for _ in projectName
    let parsedProjectName = projectName.replace(/\s/g, '_');

    const data = {
        projectName: parsedProjectName,
        ipSubrange: validIpSubrange.toString(),
        protocol: protocol,
        masterNodes: parseInt(masterNodes),
        slaveNodes: parseInt(slaveNodes)
    };
    fetch('/api/networks/', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(data)
    })
        .then(response => {
            return response.json();
        })
        .then(data => {
            if (data.status === 200) {
                console.log('Network created and saved successfully: ', data);
                window.location.href = 'networks/' + parsedProjectName; // Redirect after saving
            } else {
                console.error('Error creating the scenario: ', data.error);
                alert("Error creating the scenario: " + data.error);
            }
        })
        .catch(error => {
            alert("Error creating the scenario: " + error);
            console.log('Error creating the scenario: ', error);
        });
}

function cancelForm() {
    document.getElementById('scenario-form').reset();
    document.getElementById('form-container').style.display = 'none';
    document.getElementById('newScenarioButton').classList.remove('active');
}

function parseIpNetwork(subnet) {
    let ip = false;
    try {
        let mask = subnet.split('/')[1];
        ip = ipaddr.IPv4.networkAddressFromCIDR(subnet);
        ip = ip.toString() + '/' + mask;
    } catch (e) {
        return false;
    }
    return ip;
}

function getNextIpInSubnet(ipAddress, subnet) {
    const ip = ipaddr.parse(ipAddress);
    const subnetParsed = ipaddr.parseCIDR(subnet);

    // Verify if the IP is in the given subnet
    if (ip.match(subnetParsed)) {
        const nextIp = ip.toByteArray();

        // Increment IP
        for (let i = nextIp.length - 1; i >= 0; i--) {
            if (nextIp[i] < 255) {
                nextIp[i]++;  // Least significant byte increase
                break;
            }
            nextIp[i] = 0; // Reset on 255
        }
        return ipaddr.fromByteArray(nextIp).toString();
    } else {
        throw new Error("The IP does not belong to the specified subnet.");
    }
}

function fetchScenarios() {
    fetch('/api/networks/')
        .then(response => response.json())
        .then(data => {
            const scenarioList = document.getElementById('scenario-list');
            scenarioList.innerHTML = ''; // Clear previous content
            data.forEach(scenario => {
                const scenarioItem = document.createElement('div');
                scenarioItem.textContent = scenario;
                scenarioList.appendChild(scenarioItem);
                scenarioItem.classList.add('scenario-item');
                scenarioItem.addEventListener('click', () => loadSelectedScenario(scenario));
                scenarioList.appendChild(scenarioItem);
            });
        })
        .catch(error => {
            console.error('Error fetching scenarios:', error);
        });
}

function loadSelectedScenario(scenario) {
    window.location.href = 'networks/' + scenario;
}