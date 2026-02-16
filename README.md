# ICSCommEmulator

Herramienta en Go para emular comunicaciones ICS (Modbus, DNP3 e IEC104),
editar escenarios desde una UI web y ejecutar simulaciones con Docker Compose
capturando trafico en PCAP con `tcpdump`. Tambien soporta ataques definidos en
`web/static/attacks/attacks.json` y genera etiquetas en `*.pcap.labels.json`.

## Requisitos

- Go 1.24.x (ver `go.mod`).
- Docker Engine y Docker Compose v2 (plugin).
- `tcpdump` (para capturar trafico).

Opcional (solo si necesitas ejecutar utilidades Python o el stack DNP3 fuera
de Docker):
- Python 3 y `pip`, con dependencias en `requirements.txt` (incluye `dnp3-python`).

## Instalacion

En Linux puedes usar el script `install.sh` como referencia. Para una
instalacion manual basica:

```
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io \
  docker-buildx-plugin docker-compose-plugin tcpdump
```

Si planeas usar los componentes Python:

```
pip install -r requirements.txt
```

Nota: `tcpdump` necesita permisos. Puedes ejecutar el servidor con `sudo` o
dar capacidades al binario (`setcap cap_net_raw,cap_net_admin+eip /usr/bin/tcpdump`).

## Ejecucion

Servidor web (por defecto en `127.0.0.1:8080`):

```
go run ./cmd/icscommemulator
```

Flags disponibles:

- `-host` (default `127.0.0.1`)
- `-port` (default `8080`)
- `-log` (`DEBUG|INFO|WARNING|ERROR`)
- `-config` (ruta a YAML/JSON para importar escenarios al inicio)

Ejemplo con importacion de escenarios:

```
go run ./cmd/icscommemulator -config ./scenarios/config.yaml
```

La UI queda en `http://127.0.0.1:8080` y los PCAP se guardan en `outputs/`.

Variables de entorno utiles:

- `CORS_ALLOW_ORIGINS`: lista separada por comas para CORS.
- `ICS_ATTACKS_CATALOG_PATH`: ruta alternativa al catalogo de ataques.

## Tests

Ejecutar toda la suite Go:

```
go test ./...
```

Tests Python del paquete DNP3 (solo si tienes dependencias Python):

```
cd lib/dnp3-python
python3 -m pytest -q tests
```

Alternativas con Make:

```
make test-all
make test-unit
```

## Licencia

GNU GPLv3. Ver `LICENSE`.