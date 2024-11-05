"""
DockerComposeGenerator is a class to generate Docker Compose files programmatically.

Imports:
    - subprocess: To run subprocesses, used for validating the Docker Compose file.
    - yaml: To handle YAML file operations.
    - ipaddress: To handle IP address manipulations.
    - os: To handle file system operations.

Classes:
    - DockerComposeGenerator: Main class to generate and validate Docker Compose files.

Methods:
    - __init__(self, protocol: str, file_path: str, config_path: str): Initializes the generator with protocol, file path, and config path.
    - add_network(self, name: str, range: str): Adds a network to the Docker Compose configuration.
    - add_node(self, role: str, index: int, ip: str = None, mac: str = None, dependencies: dict[str, list[int]] = None): Adds a node (service) to the Docker Compose configuration.
    - generate(self): Generates the Docker Compose YAML file.
    - validate(self) -> bool: Validates the generated Docker Compose file.
    - validate_file(file_path: str) -> bool: Static method to validate a given Docker Compose file.
    - _validate_file(file_path: str) -> bool: Static helper method to validate a given Docker Compose file using Docker Compose CLI.

Usage:
    generator = DockerComposeGenerator("3.8", "modbus", "docker-compose.yml")
    generator.add_network("ics_network", "172.28.0.0/16")
    for i in range(10):
        generator.add_node("master", i)
    for i in range(20):
        generator.add_node("slave", i)
    generator.generate()
"""

import subprocess
import yaml
import ipaddress
import os


class DockerComposeGenerator:
    """
    A class to generate Docker Compose files programmatically.

    Attributes:
        services (dict): Dictionary to store services configuration.
        networks (dict): Dictionary to store networks configuration.
        protocol (str): Protocol name used in the Docker Compose configuration.
        ip_base (ipaddress.IPv4Address): Base IP address for the network.
        path (str): Path to the Docker Compose file.
        config_path (str): Path to the configuration files.
        last_ip (ipaddress.IPv4Address): Last assigned IP address.
        used_ips (set): Set of used IP addresses.
    """

    def __init__(self, protocol: str, file_path: str, config_path: str):
        """
        Initializes the DockerComposeGenerator with protocol, file path, and config path.

        Args:
            protocol (str): Protocol name used in the Docker Compose configuration.
            file_path (str): Path to the Docker Compose file.
            config_path (str): Path to the configuration files.
        """
        self.services = {}
        self.networks = {}
        self.protocol = protocol
        self.ip_base = None
        self.path = file_path
        self.config_path = config_path
        self.last_ip = None

    def add_network(self, name: str, range: str):
        """
        Adds a network to the Docker Compose configuration.

        Args:
            name (str): Name of the network.
            range (str): IP range for the network in CIDR notation.
        """
        self.networks[name] = {"name": name, "ipam": {"config": [{"subnet": range}]}}
        self.ip_base = ipaddress.ip_network(range).network_address
        self.last_ip = self.ip_base + 1

    def add_node(
        self,
        role: str,
        index: int,
        ip: str = None,
        mac: str = None,
        dependencies: dict[str, list[int]] = None,
    ):
        """
        Adds a node (service) to the Docker Compose configuration.

        Args:
            role (str): Role of the node (e.g., 'master', 'slave').
            index (int): Index of the node.
            ip (str, optional): IP address of the node. Defaults to None.
            mac (str, optional): MAC address of the node. Defaults to None.
            dependencies (dict, optional): Dependencies of the node. Defaults to None.
        """
        if not ip:
            self.last_ip += 1
            ip = self.last_ip
        else:
            ip = ipaddress.ip_address(ip)

        node = {
            "build": {
                "context": f"./protocols/{self.protocol}/{role}",
                "dockerfile": f"Dockerfile.{role}",
            },
            "image": f"{self.protocol}_{role}_image",
            "container_name": f"{self.protocol}_{role}_container_{index}",
            "volumes": [
                f'{self.config_path}/{role}s/{index}/{role}.{"yaml" if role=="slave" else "csv"}:/app/{role}.{"yaml" if role=="slave" else "csv"}:ro'
            ],
            "networks": {list(self.networks.keys())[0]: {"ipv4_address": str(ip)}},
            "environment": ["PYTHONUNBUFFERED=1"],
        }

        if role == "master":
            node["expose"] = ["502"]

        if role == "slave":
            node["healthcheck"] = {
                "test": ["CMD-SHELL", "test -f /app/app_running.lock"],
                "interval": "10s",
                "timeout": "5s",
                "retries": 3,
                "start_period": "10s",
            }

        if dependencies:
            node["depends_on"] = {}
            for rol, i in dependencies.items():
                for j in i:
                    node["depends_on"][f"{self.protocol}_{rol}_{j}"] = {
                        "condition": "service_healthy"
                    }

        if mac:
            node["networks"][list(self.networks.keys())[0]]["mac_address"] = mac

        self.services[f"{self.protocol}_{role}_{index}"] = node

    def generate(self):
        """
        Generates the Docker Compose YAML file.
        """
        with open(self.path, "w") as file:
            yaml.dump(
                {
                    "services": self.services,
                    "networks": self.networks,
                },
                file,
            )

    def validate(self) -> bool:
        """
        Validates the generated Docker Compose file.

        Returns:
            bool: True if the Docker Compose file is valid, False otherwise.
        """
        exists = os.path.exists(self.path)
        if not exists:
            self.generate()
        res = self._validate_file(self.path)
        if not exists:
            os.remove(self.path)
        return res

    @staticmethod
    def validate_file(file_path: str) -> bool:
        """
        Static method to validate a given Docker Compose file.

        Args:
            file_path (str): Path to the Docker Compose file.

        Returns:
            bool: True if the Docker Compose file is valid, False otherwise.
        """
        return DockerComposeGenerator._validate_file(file_path)

    @staticmethod
    def _validate_file(file_path: str) -> bool:
        """
        Static helper method to validate a given Docker Compose file using Docker Compose CLI.

        Args:
            file_path (str): Path to the Docker Compose file.

        Returns:
            bool: True if the Docker Compose file is valid, False otherwise.
        """
        result = subprocess.run(
            ["docker", "compose", "-f", file_path, "config"],
            capture_output=True,
            text=True,
        )
        return result.returncode == 0


if __name__ == "__main__":
    generator = DockerComposeGenerator("3.8", "modbus", "docker-compose.yml")
    generator.add_network("ics_network", "172.28.0.0/16")
    for i in range(10):
        generator.add_node("master", i)
    for i in range(20):
        generator.add_node("slave", i)
    generator.generate()
