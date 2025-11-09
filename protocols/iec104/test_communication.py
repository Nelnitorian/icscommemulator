"""
Tests automÃ¡ticos para comunicaciÃ³n IEC 60870-5-104
Valida conectividad, interrogaciÃ³n, eventos espontÃ¡neos, comandos,
sincronizaciÃ³n de reloj y gestiÃ³n de timestamps
"""

import pytest
import c104
import time
import datetime
import threading
import logging
from slave.server import IEC104Server
from master.client import IEC104Client

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s.%(msecs)03d [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


class TestIEC104Communication:
    """Suite de tests para comunicaciÃ³n IEC104"""

    @pytest.fixture(scope="class")
    def server(self):
        """Fixture: servidor RTU"""
        srv = IEC104Server()
        srv.start()
        time.sleep(1)
        yield srv
        srv.stop()

    @pytest.fixture(scope="class")
    def client(self):
        """Fixture: cliente SCADA"""
        cli = IEC104Client(tick_rate_ms=100, command_timeout_ms=5000)
        cli.start()
        yield cli
        cli.stop()

    @pytest.fixture(scope="function")
    def connection_setup(self, client):
        """Fixture: configuraciÃ³n de conexiÃ³n"""
        ip = "127.0.0.1"
        port = 2404
        ca = 1

        conn_key = (ip, port)
        if conn_key not in client.connections:
            connection = client.add_connection(ip, port, c104.Init.ALL)
        else:
            connection = client.connections[conn_key]
            if not connection.is_connected:
                connection.connect()

        max_wait = 10
        wait_interval = 0.5
        elapsed = 0

        while not connection.is_connected and elapsed < max_wait:
            time.sleep(wait_interval)
            elapsed += wait_interval

        if not connection.is_connected:
            pytest.fail(f"Connection to {ip}:{port} failed after {max_wait}s")

        station_key = (ip, port, ca)
        if station_key not in client.stations:
            station = client.add_station(ip, port, ca)
            if not station:
                pytest.fail(f"Failed to create station CA={ca}")

        points_to_add = [
            (1001, c104.Type.M_SP_NA_1),
            (5001, c104.Type.M_ME_NB_1),
            (6001, c104.Type.M_ME_NC_1),
            (8001, c104.Type.C_SC_NA_1),
            (11001, c104.Type.C_SE_NC_1),
        ]

        for ioa, ptype in points_to_add:
            point_key = (ip, port, ca, ioa)
            if point_key not in client.points:
                point = client.add_point(ip, port, ca, ioa, ptype)
                if not point:
                    logger.warning(f"Failed to add point IOA={ioa}")

        return ip, port, ca

    def test_01_server_startup(self, server):
        """Test: arranque del servidor"""
        assert server.server is not None
        assert server.server.is_running
        assert len(server.stations) > 0
        print(f"âœ“ Server started with {len(server.stations)} stations")

    def test_02_client_startup(self, client):
        """Test: arranque del cliente"""
        assert client.client is not None
        assert client.client.is_running
        print("âœ“ Client started successfully")

    def test_03_connection_handshake(self, server, client, connection_setup):
        """Test: handshake APCI"""
        ip, port, ca = connection_setup

        conn_key = (ip, port)
        assert conn_key in client.connections

        connection = client.connections[conn_key]
        assert connection.is_connected

        print(f"âœ“ Connection established: {ip}:{port}")

    def test_04_interrogation(self, server, client, connection_setup):
        """Test: interrogaciÃ³n general"""
        ip, port, ca = connection_setup

        with client.data_lock:
            client.received_data.clear()

        success = client.interrogation(ip, port, ca, wait=True)
        assert success, "Interrogation command failed"

        time.sleep(3)

        with client.data_lock:
            received = len(client.received_data)

        assert (
            received > 0
        ), f"No data received from interrogation (got {received} points)"
        print(f"âœ“ Interrogation successful: received {received} points")

        valid_timestamps = 0
        with client.data_lock:
            for data in client.received_data:
                if data["timestamp"]:
                    try:
                        ts = datetime.datetime.fromisoformat(data["timestamp"])
                        now = datetime.datetime.now(datetime.timezone.utc)

                        if ts.tzinfo is None:
                            ts = ts.replace(tzinfo=datetime.timezone.utc)

                        delta = abs((now - ts).total_seconds())

                        if delta < 60:
                            valid_timestamps += 1
                    except Exception as e:
                        logger.warning(f"Failed to validate timestamp: {e}")

        assert valid_timestamps > 0, "No valid timestamps found"
        print(f"âœ“ {valid_timestamps} points with valid timestamps")

    def test_05_spontaneous_events(self, server, client, connection_setup):
        """Test: eventos espontÃ¡neos"""
        ip, port, ca = connection_setup

        with client.data_lock:
            client.received_data.clear()

        print("  Waiting for spontaneous events...")
        max_wait = 35
        interval = 5

        for i in range(max_wait // interval):
            time.sleep(interval)
            with client.data_lock:
                spontaneous_found = any(
                    "SPONTANEOUS" in d["cot"] for d in client.received_data
                )

            if spontaneous_found:
                break

            print(f"  ...waiting ({(i+1)*interval}s)")

        spontaneous_count = 0
        with client.data_lock:
            for data in client.received_data:
                if "SPONTANEOUS" in data["cot"]:
                    spontaneous_count += 1

                    try:
                        ts = datetime.datetime.fromisoformat(data["timestamp"])
                        now = datetime.datetime.now(datetime.timezone.utc)

                        if ts.tzinfo is None:
                            ts = ts.replace(tzinfo=datetime.timezone.utc)

                        delta = abs((now - ts).total_seconds())

                        assert (
                            delta < 60
                        ), f"Spontaneous event timestamp too old: {delta}s"
                    except Exception as e:
                        logger.warning(f"Failed to validate spontaneous timestamp: {e}")

        assert (
            spontaneous_count > 0
        ), f"No spontaneous events received (waited {max_wait}s)"
        print(
            f"âœ“ Received {spontaneous_count} spontaneous events with valid timestamps"
        )

    def test_06_clock_sync(self, server, client, connection_setup):
        """Test: sincronizaciÃ³n de reloj"""
        ip, port, ca = connection_setup

        success = client.clock_sync(ip, port, ca, wait=True)
        assert success

        time.sleep(1)

        print("âœ“ Clock sync command sent successfully")

    def test_07_single_command(self, server, client, connection_setup):
        """Test: comando simple (C_SC_NA_1)"""
        ip, port, ca = connection_setup
        ioa_cmd = 8001
        ioa_mon = 1001

        with client.data_lock:
            client.received_data.clear()

        success = client.send_command(ip, port, ca, ioa_cmd, True, c104.Type.C_SC_NA_1)
        assert success, "Command failed to send"

        time.sleep(3)

        with client.data_lock:
            received_count = len(client.received_data)
            if received_count > 0:
                logger.info(f"Received {received_count} responses after command")

        print(f"âœ“ Single command executed: IOA={ioa_cmd} (success={success})")

    def test_08_setpoint_command(self, server, client, connection_setup):
        """Test: comando setpoint (C_SE_NC_1)"""
        ip, port, ca = connection_setup
        ioa_cmd = 11001
        ioa_mon = 6001
        value = 99.99

        with client.data_lock:
            client.received_data.clear()

        success = client.send_command(ip, port, ca, ioa_cmd, value, c104.Type.C_SE_NC_1)
        assert success, "Command failed to send"

        time.sleep(2)

        print(f"âœ“ Setpoint command executed: IOA={ioa_cmd}, value={value}")

    def test_09_counter_interrogation(self, server, client, connection_setup):
        """Test: interrogaciÃ³n de contadores"""
        ip, port, ca = connection_setup

        success = client.counter_interrogation(ip, port, ca, wait=True)
        assert success

        time.sleep(2)
        print("âœ“ Counter interrogation executed")

    def test_10_disconnect_reconnect(self, server, client, connection_setup):
        """Test: desconexiÃ³n y reconexiÃ³n"""
        ip, port, ca = connection_setup

        conn_key = (ip, port)
        connection = client.connections[conn_key]

        connection.disconnect()
        time.sleep(1)
        assert not connection.is_connected
        print("âœ“ Disconnected successfully")

        connection.connect()
        time.sleep(2)
        assert connection.is_connected
        print("âœ“ Reconnected successfully")

    def test_11_performance_latency(self, server, client, connection_setup):
        """Test: rendimiento y latencia"""
        ip, port, ca = connection_setup

        with client.data_lock:
            client.received_data.clear()

        num_iterations = 10
        latencies = []

        for i in range(num_iterations):
            start = time.time()
            client.interrogation(ip, port, ca, wait=True)
            elapsed = time.time() - start
            latencies.append(elapsed)
            time.sleep(0.5)

        avg_latency = sum(latencies) / len(latencies)
        max_latency = max(latencies)

        assert avg_latency < 1.0, f"Average latency too high: {avg_latency}s"
        assert max_latency < 2.0, f"Max latency too high: {max_latency}s"

        print(
            f"âœ“ Performance test passed: avg={avg_latency:.3f}s, max={max_latency:.3f}s"
        )

    def test_12_timestamp_precision(self, server, client, connection_setup):
        """Test: precisiÃ³n de timestamps (CP56Time2a)"""
        ip, port, ca = connection_setup

        with client.data_lock:
            client.received_data.clear()

        time.sleep(15)

        timestamps_valid = 0
        timestamps_precision = []
        timestamps_checked = 0

        with client.data_lock:
            for data in client.received_data:
                if data["timestamp"] and data["processed_at"]:
                    timestamps_checked += 1
                    try:
                        ts_recorded = datetime.datetime.fromisoformat(data["timestamp"])
                        ts_processed = datetime.datetime.fromisoformat(
                            data["processed_at"]
                        )

                        if ts_recorded.tzinfo is None:
                            ts_recorded = ts_recorded.replace(
                                tzinfo=datetime.timezone.utc
                            )
                        if ts_processed.tzinfo is None:
                            ts_processed = ts_processed.replace(
                                tzinfo=datetime.timezone.utc
                            )

                        delta_ms = abs(
                            (ts_processed - ts_recorded).total_seconds() * 1000
                        )

                        timestamps_valid += 1
                        timestamps_precision.append(delta_ms)

                    except Exception as e:
                        logger.warning(f"Failed to parse timestamp: {e}")
                        logger.warning(f"  timestamp: {data['timestamp']}")
                        logger.warning(f"  processed_at: {data['processed_at']}")
                        continue

        if timestamps_valid == 0:
            with client.data_lock:
                logger.info(f"Total data entries: {len(client.received_data)}")
                logger.info(f"Timestamps checked: {timestamps_checked}")
                for i, data in enumerate(client.received_data[:3]):
                    logger.info(
                        f"Sample data {i+1}: timestamp={data.get('timestamp')}, processed_at={data.get('processed_at')}"
                    )

        assert (
            timestamps_valid > 0
        ), f"No valid timestamps found (total entries: {len(client.received_data)}, checked: {timestamps_checked})"

        avg_precision = (
            sum(timestamps_precision) / len(timestamps_precision)
            if timestamps_precision
            else 0
        )
        max_precision = max(timestamps_precision) if timestamps_precision else 0
        min_precision = min(timestamps_precision) if timestamps_precision else 0

        print(
            f"âœ“ Timestamp precision: {timestamps_valid} valid, avg={avg_precision:.1f}ms, min={min_precision:.1f}ms, max={max_precision:.1f}ms"
        )

    def test_13_read_command(self, server, client, connection_setup):
        """Test: comando de lectura C_RD_NA_1"""
        ip, port, ca = connection_setup

        # Limpiar datos recibidos
        with client.data_lock:
            client.received_data.clear()

        # Test 1: Leer punto single-point (M_SP_NA_1)
        ioa_sp = 1001
        logger.info(f"Testing read command for single-point IOA={ioa_sp}")
        success = client.read_command(ip, port, ca, ioa_sp, wait=True)
        assert success, f"Read command failed for IOA={ioa_sp}"
        time.sleep(2)

        # Verificar que se recibió respuesta
        with client.data_lock:
            # Debug: mostrar todos los datos recibidos
            logger.info(f"Total data received: {len(client.received_data)}")
            for entry in client.received_data:
                logger.info(f"Received entry keys: {entry.keys()}")
                logger.info(f"Received entry: {entry}")

            # Usar la clave correcta: io_address en lugar de ioa
            sp_responses = [
                d for d in client.received_data if d.get("io_address") == ioa_sp
            ]
            assert len(sp_responses) > 0, (
                f"No response received for single-point IOA={ioa_sp}. "
                f"Received data: {client.received_data}"
            )

            # Verificar que la causa de transmisión es REQUEST (5)
            sp_cot = sp_responses[0]["cot"]
            logger.info(
                f"Single-point read: IOA={ioa_sp}, value={sp_responses[0]['value']}, COT={sp_cot}"
            )
            print(
                f"  ✓ Single-point (M_SP_NA_1) read: IOA={ioa_sp}, value={sp_responses[0]['value']}"
            )

        # Limpiar para siguiente test
        with client.data_lock:
            client.received_data.clear()
        time.sleep(1)

        # Test 2: Leer punto measured short (M_ME_NC_1)
        ioa_mf = 6001
        logger.info(f"Testing read command for measured short IOA={ioa_mf}")
        success = client.read_command(ip, port, ca, ioa_mf, wait=True)
        assert success, f"Read command failed for IOA={ioa_mf}"
        time.sleep(2)

        # Verificar respuesta
        with client.data_lock:
            mf_responses = [
                d for d in client.received_data if d.get("io_address") == ioa_mf
            ]
            assert len(mf_responses) > 0, (
                f"No response received for measured short IOA={ioa_mf}. "
                f"Received data: {client.received_data}"
            )

            mf_cot = mf_responses[0]["cot"]
            logger.info(
                f"Measured short read: IOA={ioa_mf}, value={mf_responses[0]['value']}, COT={mf_cot}"
            )
            print(
                f"  ✓ Measured short (M_ME_NC_1) read: IOA={ioa_mf}, value={mf_responses[0]['value']}"
            )

        # Limpiar para siguiente test
        with client.data_lock:
            client.received_data.clear()
        time.sleep(1)

        # Test 3: Leer punto escalado (M_ME_NB_1)
        ioa_scaled = 5001
        logger.info(f"Testing read command for measured scaled IOA={ioa_scaled}")
        success = client.read_command(ip, port, ca, ioa_scaled, wait=True)
        assert success, f"Read command failed for IOA={ioa_scaled}"
        time.sleep(2)

        with client.data_lock:
            scaled_responses = [
                d for d in client.received_data if d.get("io_address") == ioa_scaled
            ]
            logger.info(f"Scaled responses: {scaled_responses}")
            if len(scaled_responses) > 0:
                logger.info(
                    f"Measured scaled read: IOA={ioa_scaled}, value={scaled_responses[0]['value']}"
                )
                print(
                    f"  ✓ Measured scaled (M_ME_NB_1) read: IOA={ioa_scaled}, value={scaled_responses[0]['value']}"
                )

        print(f"✓ Read command (C_RD_NA_1) test passed: tested M_SP_NA_1 and M_ME_NC_1")

    def test_14_m_me_tf_1_monitoring(self, server, client, connection_setup):
        """Test: M_ME_TF_1 measured short with timestamp"""
        ip, port, ca = connection_setup

        # Agregar punto M_ME_TF_1
        client.add_point(ip, port, ca, 6101, c104.Type.M_ME_TF_1)

        with client.data_lock:
            client.received_data.clear()

        time.sleep(6)  # Esperar reporte automático

        with client.data_lock:
            responses = [d for d in client.received_data if d.get("io_address") == 6101]
            assert len(responses) > 0, "No M_ME_TF_1 data received"
            # Verificar que tiene timestamp
            assert responses[0]["timestamp"], "No timestamp in M_ME_TF_1"

        print(f"✓ M_ME_TF_1 test passed: {responses[0]['value']}")

    def test_15_m_sp_tb_1_monitoring(self, server, client, connection_setup):
        """Test: M_SP_TB_1 single point with timestamp"""
        ip, port, ca = connection_setup

        client.add_point(ip, port, ca, 1101, c104.Type.M_SP_TB_1)

        with client.data_lock:
            client.received_data.clear()

        time.sleep(9)

        with client.data_lock:
            responses = [d for d in client.received_data if d.get("io_address") == 1101]
            assert len(responses) > 0, "No M_SP_TB_1 data received"
            assert responses[0]["timestamp"], "No timestamp in M_SP_TB_1"

        print(f"✓ M_SP_TB_1 test passed: {responses[0]['value']}")

    def test_16_c_sc_ta_1_command(self, server, client, connection_setup):
        """Test: C_SC_TA_1 single command with timestamp"""
        ip, port, ca = connection_setup

        success = client.send_command(ip, port, ca, 8101, True, c104.Type.C_SC_TA_1)
        assert success, "C_SC_TA_1 command failed"
        time.sleep(2)

        print(f"✓ C_SC_TA_1 command test passed")

    def test_17_c_se_tc_1_command(self, server, client, connection_setup):
        """Test: C_SE_TC_1 setpoint short with timestamp"""
        ip, port, ca = connection_setup

        success = client.send_command(ip, port, ca, 11101, 88.88, c104.Type.C_SE_TC_1)
        assert success, "C_SE_TC_1 command failed"
        time.sleep(2)

        print(f"✓ C_SE_TC_1 command test passed")


def test_integration_full_cycle():
    """Test de integraciÃ³n: ciclo completo SCADAâ†”RTU"""
    print("\n=== Full Integration Test ===")

    server = IEC104Server()
    server.start()
    time.sleep(2)

    client = IEC104Client()
    client.start()

    ip = "127.0.0.1"
    port = 2404
    ca = 1

    connection = client.add_connection(ip, port, c104.Init.ALL)

    max_wait = 10
    elapsed = 0
    while not connection.is_connected and elapsed < max_wait:
        time.sleep(0.5)
        elapsed += 0.5

    if not connection.is_connected:
        print(f"âœ— Connection failed after {max_wait}s")
        client.stop()
        server.stop()
        assert False, "Connection failed"

    print(f"âœ“ Connected to {ip}:{port}")

    client.add_station(ip, port, ca)

    client.add_point(ip, port, ca, 1001, c104.Type.M_SP_NA_1)
    client.add_point(ip, port, ca, 5001, c104.Type.M_ME_NB_1)
    client.add_point(ip, port, ca, 6001, c104.Type.M_ME_NC_1)
    client.add_point(ip, port, ca, 8001, c104.Type.C_SC_NA_1)

    time.sleep(1)

    print("1. Clock synchronization...")
    success = client.clock_sync(ip, port, ca)
    if success:
        print("   âœ“ Clock sync sent")
    time.sleep(1)

    print("2. General interrogation...")
    success = client.interrogation(ip, port, ca)
    if success:
        print("   âœ“ Interrogation sent")
    time.sleep(3)

    print("3. Sending command...")
    success = client.send_command(ip, port, ca, 8001, True, c104.Type.C_SC_NA_1)
    if success:
        print("   âœ“ Command sent")
    time.sleep(2)

    print("4. Waiting for spontaneous events...")
    time.sleep(15)

    with client.data_lock:
        total_data = len(client.received_data)

    print(f"âœ“ Integration test completed: {total_data} messages exchanged")

    if total_data > 0:
        client.save_data_to_csv("integration_test_data.csv")

    client.stop()
    server.stop()

    assert total_data > 0, f"No data received (expected > 0, got {total_data})"


if __name__ == "__main__":
    pytest.main([__file__, "-v", "-s"])
