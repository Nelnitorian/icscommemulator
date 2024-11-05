from .cytoscape_adapter import parse_cytoscape_json
import os
import json
import yaml
from typing import Any

# SCENARIO_ROOT_FOLDER = os.path.join("web", "scenarios")
SCENARIO_ROOT_FOLDER = "scenarios"
os.makedirs(SCENARIO_ROOT_FOLDER, exist_ok=True)


def save_scenario(name: str, raw_data: dict[str, Any]):
    scenario_folder = os.path.join(SCENARIO_ROOT_FOLDER, name)
    os.makedirs(scenario_folder, exist_ok=True)

    json_file = os.path.join(scenario_folder, "config.json")
    yaml_file = os.path.join(scenario_folder, "config.yaml")

    parsed_data = parse_cytoscape_json(raw_data)

    with open(json_file, "w") as f:
        json.dump(raw_data, f)
    with open(yaml_file, "w") as f:
        f.write(parsed_data)


def get_created_scenarios() -> list[str]:
    return os.listdir(SCENARIO_ROOT_FOLDER)


def get_cytoscape_scenario(name: str) -> dict[str, Any]:
    scenario_folder = os.path.join(SCENARIO_ROOT_FOLDER, name)
    json_file = os.path.join(scenario_folder, "config.json")
    with open(json_file, "r") as f:
        return json.load(f)


def check_scenario_exists(name: str) -> bool:
    return os.path.exists(os.path.join(SCENARIO_ROOT_FOLDER, name))


def get_python_scenario(name: str) -> dict[str, Any]:
    scenario_folder = os.path.join(SCENARIO_ROOT_FOLDER, name)
    yaml_file = os.path.join(scenario_folder, "config.yaml")
    with open(yaml_file, "r") as f:
        return yaml.safe_load(f)
