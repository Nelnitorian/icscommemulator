import yaml
from enum import Enum
from typing import Any
import ipaddress


class LEVEL(Enum):
    ERROR = 1
    WARNING = 2


class PROTOCOL(Enum):
    MODBUS = "modbus"


def validate_cytoscape_scenario(data: dict[str, Any], level: LEVEL) -> list[str]:
    logs = []
    # Check if all ids are different
    if level.value >= LEVEL.ERROR.value:
        node_ids = [node["data"]["id"] for node in data["nodes"]]
        if len(node_ids) != len(set(node_ids)):
            # Find duplicated node IDs
            from collections import Counter

            duplicates = [
                item for item, count in Counter(node_ids).items() if count > 1
            ]

            # Print duplicated node IDs
            logs.append(
                f"[ERROR] All node IDs are not unique. Duplicates: {duplicates}"
            )

        # Check if the edges are made from nodes with different roles
        node_roles = {
            node["data"]["id"]: node["data"]["role"] for node in data["nodes"]
        }
        for edge in data["edges"]:
            source_role = node_roles[edge["data"]["source"]]
            target_role = node_roles[edge["data"]["target"]]
            if source_role == target_role:
                logs.append(
                    f"[ERROR] Edge {edge['data']['id']} connects nodes with the same role ({source_role})"
                )

        # Check if there are ips out of range
        ip_network = ipaddress.ip_network(data["ip_network"])
        for node in data["nodes"]:
            ip = ipaddress.ip_address(node["data"]["ip"])
            if ip not in ip_network:
                logs.append(
                    f"[ERROR] IP {ip} of node {node['data']['id']} is out of range"
                )

    if level.value >= LEVEL.WARNING.value:
        # Check if there is no slave without any defined register
        for node in data["nodes"]:
            if node["data"]["role"] == "slave":
                if not any(
                    node["data"][key]["values"]
                    for key in [
                        "holding_registers",
                        "coils",
                        "discrete_inputs",
                        "input_registers",
                    ]
                ):
                    logs.append(
                        f"[WARNING] Slave node {node['data']['id']} has no defined registers"
                    )
        # Check if there is no edge with an empty "messages" field and start_address >= 0
        for edge in data["edges"]:
            if (
                not all(
                    edge["data"][key]
                    for key in [
                        "timestamp",
                        "function_code",
                        "start_address",
                        "input_registers",
                    ]
                )
                and not edge["data"]["start_address"] >= 0
            ):
                logs.append(
                    f"[WARNING] Communication {edge['data']['source']} → {edge['data']['target']} has empty values."
                )

        # Extract source and target IDs from the links
        linked_node_ids = set()
        for edge in data["edges"]:
            linked_node_ids.add(edge["data"]["source"])
            linked_node_ids.add(edge["data"]["target"])

        # Find nodes without links
        nodes_without_links = [
            node_id for node_id in node_ids if node_id not in linked_node_ids
        ]

        # Print nodes without links if any are found
        for node in nodes_without_links:
            logs.append(f"[WARNING] Node without link: {node}")
    return logs


def parse_cytoscape_json(data: dict[str, Any]) -> str:
    yaml_data = {"nodes": []}

    yaml_data["protocol"] = data["protocol"]
    yaml_data["ip_network"] = data["ip_network"]

    # Create a dictionary to store messages for master nodes
    messages_dict = {
        node["data"]["id"]: []
        for node in data["nodes"]
        if node["data"]["role"] == "master"
    }

    # Parse edges to generate messages for master nodes
    for edge in data["edges"]:
        for message in edge["data"]["messages"]:
            target_node_data = find_first_matching_node(
                data["nodes"], edge["data"]["target"]
            )["data"]
            message = {
                # "recurrent": edge["data"]["recurrent"],
                "timestamp": message["timestamp"],
                "recurrent": message["recurrent"],
                "interval": message["interval"],
                "ip": target_node_data["ip"],  # Assuming default IP, modify as needed
                "port": target_node_data[
                    "port"
                ],  # Assuming default port, modify as needed
                "slave_id": target_node_data["slave_id"],
                "function_code": message["function_code"],
                "start_address": message["start_address"],
                "count": message.get("count", 0),
                "values": message.get("values", []),
            }
            messages_dict[edge["data"]["source"]].append(message)

    # Add nodes to yaml_data
    for node in data["nodes"]:
        node_data = node["data"]
        if node_data["role"] == "master":
            node_data["messages"] = messages_dict[node_data["id"]]
        yaml_data["nodes"].append(node_data)

    # Convert to YAML
    yaml_output = yaml.dump(yaml_data, default_flow_style=False)
    return yaml_output


def find_first_matching_node(
    nodes: dict[dict[str, Any]], node_id: str
) -> dict[str, Any]:
    for node in nodes:
        if node["data"]["id"] == node_id:
            return node
    return None


# Function to create master and slave nodes
def generate_network_nodes(
    proto: PROTOCOL, ip_base: ipaddress.IPv4Address, master_nodes: int, slave_nodes: int
) -> dict[str, Any]:
    nodes = []
    edges = []

    # Create master nodes
    for i in range(master_nodes):
        master_node = {
            "data": {
                "id": f"master_{i}",  # Unique ID for the master node
                "label": "data(name)",
                "role": "master",
                "name": f"master_{i}",
                "ip": str(ip_base),
            },
            "classes": "master",
        }
        ip_base += 1
        nodes.append(master_node)

    # Create slave nodes
    for i in range(slave_nodes):
        slave_node = {}
        data = {
            "id": f"slave_{i}",  # Unique ID for the slave node
            "label": "data(name)",
            "role": "slave",
            "name": f"slave_{i}",
            "ip": str(ip_base),
        }
        if proto == PROTOCOL.MODBUS.value:
            data["port"] = "502"
            data["slave_id"] = "1"
            data["comment"] = ""
            for register in [
                "holding_registers",
                "coils",
                "discrete_inputs",
                "input_registers",
            ]:
                data[register] = {}
                data[register]["type"] = "sequential"
                data[register]["values"] = ""
            data["identity"] = {
                "major_minor_revision": "",
                "model_name": "",
                "product_code": "",
                "product_name": "",
                "user_application_name": "",
                "vendor_name": "",
                "vendor_url": "",
            }
        ip_base += 1
        slave_node["data"] = data
        slave_node["classes"] = "slave"
        nodes.append(slave_node)

    return nodes, edges
