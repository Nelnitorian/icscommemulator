import os
import signal
import unittest
import threading
import time

from protocols.modbus.master.master import ModbusMaster
from protocols.modbus.slave.slave import ModbusSlave


class TestModbusConsumer(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        """
        Initial setup for the test, creating an instance of the Modbus consumer.
        """
        cls.consumer = ModbusSlave("tests/modbus_slave.yaml", "tests/app_running.lock")
        cls.consumer_thread = threading.Thread(target=cls.consumer.start)
        cls.consumer_thread.start()
        time.sleep(2)  # Wait for the server to start

    @classmethod
    def tearDownClass(cls):
        """
        Shuts down the Modbus server after the tests have finished.
        """
        os.remove("tests/app_running.lock")
        cls.consumer.shutdown()
        cls.consumer_thread.join(timeout=3)
        if cls.consumer_thread.is_alive():
            print("Thread did not stop in time. Forcing shutdown.")
            os.kill(os.getpid(), signal.SIGTERM)

    def test_modbus_message_reception(self):
        """
        Basic test to send a Modbus request to the consumer and validate the reception.
        """
        try:
            # Create a Modbus TCP client
            client = ModbusMaster("tests/modbus_master.csv")
            client.loop()
            for response in client.responses:
                print(f"Modbus Response: {response}")
                self.assertIsNotNone(
                    response, "Modbus message failed to send or no response received."
                )
                self.assertFalse(response.isError(), "Modbus response is an error.")

        except Exception as e:
            self.fail(f"Error during Modbus message reception test: {e}")


if __name__ == "__main__":
    unittest.main()
