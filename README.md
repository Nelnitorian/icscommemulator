# ICSCommEmulator

## About the project

ICSCommEmulator is an emulation tool designed to emulate communication protocols used in Industrial Control Systems (ICS). The project aims to facilitate the testing and validation of ICS communication by providing a controlled environment where the Modbus protocol can be emulated. It uses Docker for containerization, tcpdump for network traffic capture, and Python for scripting and automation.

## Installation

For the installation process, a script `install.sh` has been included.

The Python library requirements are in the `requirements.txt` file. Installation via `pip install -r requirements.txt` can be done.

The package requirements can be found within this document or in the `install.sh` file.

> [!NOTE]
> In the installation script, the permissions for /usr/bin/tcpdump are modified. This allows it to run without sudo, as it is required by main.py.

## Dependencies

### Docker

- Docker version 27.3.1, build ce12230
- Docker Compose version v2.29.7

### Tcpdump

- tcpdump version 4.99.1
- libpcap version 1.10.1 (with TPACKET_V3)
- OpenSSL 3.0.2 15 Mar 2022

### Python

- Python 3.10.12

| Library   | Version | Usage                        |
|-----------|---------|------------------------------|
| flask     | 3.0.3   | Web framework for the API    |
| pandas    | 2.2.3   | Data manipulation            |
| pymodbus  | 3.7.2   | Modbus communication         |
| pytest    | 8.3.3   | Unit testing framework       |
| waitress  | 3.0.0   | WSGI server for Flask        |

### Frontend

As part of the frontend, the following JavaScript libraries have been used:

- [Cytoscape.js](https://unpkg.com/cytoscape@3.18.1/dist/cytoscape.min.js)
- [ipaddr.js](https://github.com/whitequark/ipaddr.js/blob/main/ipaddr.min.js)

## Usage

Once installed, the main script can be run. It will deploy a web server listening on 0.0.0.0:8080.

```python3 main.py```

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.