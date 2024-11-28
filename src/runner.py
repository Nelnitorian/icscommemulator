import datetime
import os
import shutil
import subprocess
import json
import threading
import time
import yaml
import logging
from threading import Lock

logger = logging.getLogger(__name__)
logging.basicConfig(level=logging.INFO)


class ScenarioRunner:
    _lock = Lock()
    _is_running = False
    _instance = None

    def __new__(cls, *args, **kwargs):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def __init__(self):
        return

    def config(
        self,
        docker_compose_path: str,
        simulation_time: int,
        output_file: str = None,
        config_path: str = None,
    ):
        self.file_path = docker_compose_path
        self.simulation_time = simulation_time
        self.config_path = config_path or "/tmp/ICSCommEmulator"
        self.output_folder = "outputs"
        self.output_file = os.path.join(self.output_folder, output_file)
        self.start_time = None
        self._tcpdump_process = None

    def get_docker_network_interface(self) -> list[str]:
        with open(self.file_path, "r") as file:
            yaml_file = yaml.safe_load(file)
        return list(yaml_file["networks"].keys())[0]

    def get_system_interface_name(self, docker_network_name: str) -> str:
        result = subprocess.run(
            ["docker", "network", "inspect", docker_network_name],
            capture_output=True,
            text=True,
        )
        network_data = json.loads(result.stdout)
        network_id = network_data[0]["Id"][:12]
        return "br-" + network_id

    def start_tcpdump(self, interface_name: str) -> subprocess.Popen | None:
        try:
            with open("/dev/null", "w") as stdout_file, open(
                "/dev/null", "w"
            ) as stderr_file:
                command = [
                    "tcpdump",
                    "-i",
                    interface_name,
                    "-w",
                    self.output_file,
                    "-U",
                    "-nn",
                    "not udp port 5353",  # filter mDNS
                    "and not udp port 1900",  # filter SSDP
                ]
                logger.info(
                    f"Starting tcpdump on interface '{interface_name}', saving to '{self.output_file}'"
                )
                process = subprocess.Popen(
                    command, stdout=stdout_file, stderr=stderr_file
                )
                return process
        except Exception as e:
            logger.error(f"Error starting tcpdump: {e}")
            return None

    def launch_docker_compose(self):
        self.ensure_launchable()
        logger.info("Launching docker compose...")
        result = subprocess.run(
            [
                "docker",
                "compose",
                "-f",
                self.file_path,
                "up",
                "--build",
                "-d",
                "--remove-orphans",
            ],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            logger.error(f"Failed to launch docker compose:\n{result.stderr}")
            return
        logger.info("Starting network traffic capture...")

    def ensure_launchable(self):
        with open(self.file_path, "r") as file:
            yaml_file = yaml.safe_load(file)
        network_name = list(yaml_file["networks"].keys())[0]
        subprocess.run(["docker", "network", "rm", network_name], text=True)

    def stop_docker_compose(self):
        logger.info("Stopping docker compose...")
        result = subprocess.run(
            ["docker", "compose", "-f", self.file_path, "down", "--timeout", "3"],
            capture_output=True,
            text=True,
        )
        if result.returncode != 0:
            logger.error(f"Failed to stop docker compose:\n{result.stderr}")
            return
        logger.info("Docker compose stopped.")

    def stop_tcpdump(self):
        if self._tcpdump_process:
            logger.info("Stopping tcpdump...")
            self._tcpdump_process.terminate()
            logger.info("Tcpdump stopped.")
        else:
            logger.error("Tcpdump process not found.")

    def run(self):
        with ScenarioRunner._lock:
            if ScenarioRunner._is_running:
                raise Exception("Another scenario is already running.")
            ScenarioRunner._is_running = True
            self.running = True
            try:
                self.launch_docker_compose()
                iface = self.get_system_interface_name(
                    self.get_docker_network_interface()
                )
                self._tcpdump_process = self.start_tcpdump(iface)
                self.start_time = datetime.datetime.now()
                time.sleep(self.simulation_time + 1)
                logger.info("Stopping network traffic capture...")
                self._tcpdump_process.terminate()
                self.running = False
                self.stop_docker_compose()
                self.clean_config_folder()
            except Exception as e:
                logger.error(f"Error during scenario execution: {e}")
                self.stop_docker_compose()
            finally:
                ScenarioRunner._is_running = False
                self.running = False
                self.simulation_time = None
                self._tcpdump_process = None

    def clean_config_folder(self):
        if os.path.exists(self.config_path):
            shutil.rmtree(self.config_path)

    def status(self):
        if not self.start_time or not self.running:
            logger.warning("Simulation has not started.")
            return {"error": "Simulation not started."}

        # Calculate elapsed and total time
        elapsed_time = datetime.datetime.now() - self.start_time
        elapsed_seconds = int(elapsed_time.total_seconds())
        total_seconds = self.simulation_time

        # Get the size of the pcap file
        pcap_size = (
            os.path.getsize(self.output_file) if os.path.exists(self.output_file) else 0
        )

        return {
            "elapsed_seconds": elapsed_seconds,
            "total_seconds": total_seconds,
            "pcap_size": pcap_size,  # bytes
            "running": self.running,
        }


runner: ScenarioRunner = None


def start(
    docker_compose_path: str,
    simulation_time: int,
    output_file: str,
    config_path: str = None,
) -> str:
    global runner
    if not runner:
        runner = ScenarioRunner()

    if runner._is_running:
        logger.error("A scenario is already running.")
        return
    runner.config(docker_compose_path, simulation_time, output_file, config_path)
    thread = threading.Thread(target=runner.run)
    thread.start()
    return os.path.abspath(runner.output_file)


def stop():
    global runner
    if not runner:
        logger.error("No scenario is running.")
        return
    runner.stop_docker_compose()
    runner.stop_tcpdump()


def status():
    global runner
    if not runner:
        logger.error("No scenario is running.")
        return
    return runner.status()


if __name__ == "__main__":
    start("docker-compose.yml", 10, "output.pcap")
