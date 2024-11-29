from flask import Flask, render_template, request, jsonify, abort
import json
import ipaddress

from src.cytoscape_adapter import (
    validate_cytoscape_scenario,
    generate_network_nodes,
    LEVEL,
)
from src.docker_compose_generator import DockerComposeGenerator
from src.scenario_config_generator import ScenarioConfigGenerator
from src.scenario_handler import (
    get_python_scenario,
    save_scenario,
    get_cytoscape_scenario,
    get_created_scenarios,
    check_scenario_exists,
)

from src.runner import start, stop, status


class NetworkAPI:
    def __init__(self):
        self.app = Flask(__name__)
        self.setup_routes()

    def setup_routes(self):
        self.app.add_url_rule(
            "/api/networks/", view_func=self.handle_network, methods=["GET", "POST"]
        )
        self.app.add_url_rule(
            "/api/networks/<name>",
            view_func=self.handle_network,
            methods=["GET", "PUT"],
        )
        self.app.add_url_rule("/", view_func=self.home)
        self.app.add_url_rule("/index.html", view_func=self.home)
        self.app.add_url_rule("/networks/<id>", view_func=self.network)
        self.app.add_url_rule(
            "/api/run/", view_func=self.handle_run, methods=["GET", "DELETE"]
        )
        self.app.add_url_rule(
            "/api/run/<name>", view_func=self.handle_run, methods=["POST"]
        )

    def handle_network(self, name=None):
        if request.method == "GET":
            if name:
                try:
                    scenario = get_cytoscape_scenario(name)
                    return jsonify(scenario), 200
                except:
                    return jsonify({"status": 404, "error": "Network not found"}), 404
            else:
                scenarios = get_created_scenarios()
                return jsonify(scenarios), 200
        elif request.method == "POST":
            data = request.get_json()
            name = data.get("projectName")
            ip_range = data.get("ipSubrange")
            protocol = data.get("protocol")

            if check_scenario_exists(name):
                return jsonify({"status": 400, "error": "Project already exists"}), 400

            try:
                master_nodes = int(data.get("masterNodes"))
                slave_nodes = int(data.get("slaveNodes"))
            except (TypeError, ValueError):
                return (
                    jsonify(
                        {"status": 400, "error": "Invalid masterNodes or slaveNodes"}
                    ),
                    400,
                )

            if not all([name, ip_range, protocol, master_nodes, slave_nodes]):
                return jsonify({"status": 400, "error": "Missing parameters"}), 400

            try:
                ip_network = ipaddress.ip_network(ip_range)
            except ValueError:
                return jsonify({"status": 400, "error": "Invalid IP range"}), 400

            nodes, edges = generate_network_nodes(
                protocol, ip_network.network_address + 2, master_nodes, slave_nodes
            )
            network = {
                "protocol": protocol,
                "ip_network": str(ip_network),
                "nodes": nodes,
                "edges": edges,
            }

            save_scenario(name, network)
            return (
                jsonify(
                    {"message": f"Network created and saved as {name}", "status": 200}
                ),
                200,
            )

        elif request.method == "PUT":
            data = request.get_json()
            if isinstance(data, str):
                data = json.loads(data)

            try:
                logs = validate_cytoscape_scenario(data, LEVEL.ERROR)
                if logs:
                    return jsonify({"status": 400, "error": logs}), 400
            except Exception:
                return jsonify({"status": 400, "error": "error validating json"}), 400

            save_scenario(name, data)
            return jsonify({"message": f"Scenario saved as {name}"}), 200

    def home(self):
        return render_template("index.html")

    def network(self, id):
        try:
            network_data = get_cytoscape_scenario(id)
        except Exception:
            abort(404)
        return render_template("network.html", network_data=network_data)

    def handle_run(self, name=None):
        if request.method == "POST":
            data = request.get_json()
            simulation_time = data.get("simulation_time")
            try:
                scenario = get_python_scenario(name)
            except Exception as e:
                return (
                    jsonify({"status": 404, "error": f"Scenario not found: {e}"}),
                    404,
                )
            try:
                docker_compose_path = "docker-compose.yml"
                config_path = "/tmp/ICSCommEmulator"
                self.generate_docker_compose(scenario, docker_compose_path, config_path)
                self.generate_scenario_config(scenario, config_path)
                file_path = start(
                    docker_compose_path, simulation_time, f"{name}.pcap", config_path
                )
            except Exception as e:
                return (
                    jsonify({"status": 500, "error": f"Error running scenario: {e}"}),
                    500,
                )

            return (
                jsonify(
                    {
                        "message": "Scenario running",
                        "simulation_time": simulation_time,
                        "file_path": file_path,
                    }
                ),
                200,
            )

        elif request.method == "GET":
            try:
                scenario_status = self.get_scenario_status()
            except Exception as e:
                return (
                    jsonify(
                        {"status": 500, "error": f"Error fetching scenario status: {e}"}
                    ),
                    500,
                )

            return jsonify(scenario_status), 200

        elif request.method == "DELETE":
            try:
                self.stop_scenario()
            except Exception as e:
                return (
                    jsonify({"status": 500, "error": f"Error stopping scenario: {e}"}),
                    500,
                )

            return jsonify({"message": "Scenario stopped"}), 200

    def get_scenario_status(self):
        return status()

    def stop_scenario(self):
        stop()

    def generate_scenario_config(
        self, scenario, scenario_config_path="/tmp/ICSCommEmulator"
    ):
        scg = ScenarioConfigGenerator(scenario, scenario_config_path)
        scg.generate()

    def generate_docker_compose(
        self,
        scenario,
        docker_compose_path="docker-compose.yml",
        scenario_config_path="/tmp/ICSCommEmulator",
    ):
        dcg = DockerComposeGenerator(
            scenario["protocol"], docker_compose_path, scenario_config_path
        )
        dcg.parse(scenario)

    def run(self, host="127.0.0.1", port=8080):
        from waitress import serve

        serve(self.app, host=host, port=port)


# To use this class, create an instance and call run() or import and use it in another module:
# app = NetworkAPI()
# app.run()
if __name__ == "__main__":
    app = NetworkAPI()
    app.run()
