"""
ScenarioConfigGenerator is a class to generate configuration files for a given scenario.

Imports:
    - yaml: To handle YAML file operations.
    - os: To handle file system operations.
    - numpy as np: To handle numerical operations.
    - pandas as pd: To handle data manipulation and analysis.
    - shutil: To handle high-level file operations.
    - typing: To handle type hints.

Classes:
    - ScenarioConfigGenerator: Main class to generate configuration files for masters and slaves.

Methods:
    - __init__(self, scenario: dict[str, Any], config_path: str): Initializes the generator with scenario and config path.
    - _convert_to_int(x: str) -> str | int: Static method to convert a string to an integer if possible.
    - _craft_master(self, messages: list[dict[str, Any]], i: int): Creates configuration files for master nodes.
    - _craft_slave(self, slave: dict[str, Any], i: int): Creates configuration files for slave nodes.
    - clean(self): Cleans the configuration path by removing existing files.
    - generate(self): Generates the configuration files for the scenario.

Usage:
    scenario = {
        "nodes": [
            {"role": "master", "messages": [...]},
            {"role": "slave", ...}
        ]
    }
    generator = ScenarioConfigGenerator(scenario, "config_path")
    generator.generate()
"""

import os
import shutil
from typing import Any

import numpy as np
import pandas as pd
import yaml


class ScenarioConfigGenerator:
    """
    A class to generate configuration files for a given scenario.

    Attributes:
        scenario (dict): Dictionary containing the scenario configuration.
        config_path (str): Path to the configuration files.
    """

    def __init__(self, scenario: dict[str, Any], config_path: str):
        """
        Initializes the ScenarioConfigGenerator with scenario and config path.

        Args:
            scenario (dict): Dictionary containing the scenario configuration.
            config_path (str): Path to the configuration files.
        """
        self.scenario = scenario
        self.config_path = config_path

    @staticmethod
    def _convert_to_int(x: str) -> str | int:
        """
        Static method to convert a string to an integer if possible.

        Args:
            x (str): The string to convert.

        Returns:
            str | int: The converted integer or the original string if conversion fails.
        """
        try:
            return int(x)
        except ValueError:
            return x

    def _craft_master(self, messages: list[dict[str, Any]], i: int):
        """
        Creates configuration files for master nodes.

        Args:
            messages (list[dict[str, Any]]): List of messages for the master node.
            i (int): Index of the master node.
        """
        os.makedirs(f"{self.config_path}/masters/{i}", exist_ok=True)
        df = pd.DataFrame(messages)
        if not df.empty:
            df["count"] = (
                df["count"]
                .astype(object)
                .replace(np.nan, "")
                .apply(ScenarioConfigGenerator._convert_to_int)
            )
            df.to_csv(f"{self.config_path}/masters/{i}/master.csv", index=False)
        else:
            with open(f"{self.config_path}/masters/{i}/master.csv", "w") as f:
                f.write("")

    def _craft_slave(self, slave: dict[str, Any], i: int):
        """
        Creates configuration files for slave nodes.

        Args:
            slave (dict[str, Any]): Dictionary containing the slave configuration.
            i (int): Index of the slave node.
        """
        os.makedirs(f"{self.config_path}/slaves/{i}", exist_ok=True)

        for key in ["comment", "label", "role", "name", "id"]:
            if key in slave:
                del slave[key]

        with open(f"{self.config_path}/slaves/{i}/slave.yaml", "w") as f:
            yaml.dump(slave, f)

    def clean(self):
        """
        Cleans the configuration path by removing existing files.
        """
        if os.path.exists(self.config_path):
            shutil.rmtree(self.config_path)

    def generate(self):
        """
        Generates the configuration files for the scenario.
        """
        self.clean()

        for i, node in enumerate(filter(is_master, self.scenario["nodes"])):
            self._craft_master(node["messages"], i)

        for i, node in enumerate(filter(is_slave, self.scenario["nodes"])):
            self._craft_slave(node, i)


def is_master(dic):
    return dic["role"] == "master"


def is_slave(dic):
    return dic["role"] == "slave"
