import unittest
import yaml
import os
from src.docker_compose_generator import DockerComposeGenerator


class TestDockerComposeGeneration(unittest.TestCase):
    def test_generate_docker_compose(self):
        """
        Test the generation of a Docker Compose file with a number of master and slave nodes.
        Validates the presence of all services and the correctness of the generated file.
        """
        file_path = "tests/docker-compose_test.yml"

        num_masters = 1
        num_slaves = 1

        mac_addresses = ["d8:30:82:6d:81:a3", "d8:30:82:6d:81:a4"]

        generator = DockerComposeGenerator("modbus", file_path, "/tmp/ICSCommEmulator")
        generator.add_network("ics_network", "172.28.0.0/16")
        for i in range(num_masters):
            generator.add_node(
                "master", i, dependencies={"slave": [0]}, mac=mac_addresses.pop()
            )
        for i in range(num_slaves):
            generator.add_node("slave", i, mac=mac_addresses.pop())
        generator.generate()

        with open(file_path, "r") as file:
            docker_compose = yaml.safe_load(file)

        self.assertEqual(len(docker_compose["services"]), num_masters + num_slaves)

        for i in range(num_masters):
            self.assertIn(f"modbus_master_{i}", docker_compose["services"])

        for i in range(num_slaves):
            self.assertIn(f"modbus_slave_{i}", docker_compose["services"])

        self.assertTrue(generator.validate())

        os.remove(file_path)


if __name__ == "__main__":
    unittest.main()
