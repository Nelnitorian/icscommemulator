# Compilando dnp3-python en Arch Linux

## Por qué esta guía

Intenté instalar dnp3-python con `pip install dnp3-python` en Arch con Python 3.12.11 y fue imposible. El paquete no tiene wheels para Python 3.11+ y la compilación falla por varios problemas. Este documento explica cómo lo resolví.

## Problemas que encontré

1. No hay wheels en PyPI para Python 3.11+
2. Arch usa CMake 4.0 que ya no acepta `cmake_minimum_required(VERSION 2.8)`
3. La versión de pybind11 incluida no compila con Python 3.12

## Dependencias necesarias

```bash
sudo pacman -S base-devel cmake git openssl python-pip
```

En Arch los headers de Python vienen en el paquete `python` directamente, no hay que instalar nada extra.

## Pasos para compilar

### 1. Entorno virtual

```bash
cd ~/Proyectos/rapido
mkdir dnp3
cd dnp3

python -m venv .venv
source .venv/bin/activate
pip install --upgrade pip setuptools wheel
```

### 2. Clonar el repo

```bash
git clone --recursive https://github.com/VOLTTRON/dnp3-python.git
cd dnp3-python
```

### 3. Arreglar CMakeLists.txt y .cmake

```bash
find . \( -name "CMakeLists.txt" -o -name "*.cmake" \) -type f -exec sed -i -E 's/cmake_minimum_required[[:space:]]*\(VERSION[[:space:]]+[0-9]+\.[0-9]+(\.[0-9]+)?\)/cmake_minimum_required(VERSION 3.10)/g' {} \;
```

### 5. Actualizar pybind11

La versión incluida es muy vieja y da errores con Python 3.12:
- `error: invalid use of incomplete type 'PyFrameObject'`
- `error: 'PyThreadState' has no member named 'frame'`

Hay que reemplazarla:

```bash
cd deps
mv pybind11 pybind11.old

git clone https://github.com/pybind/pybind11.git pybind11
cd pybind11
git checkout v2.13.6

cd ~/Proyectos/rapido/dnp3/dnp3-python
```

### 6. Compilar

```bash
rm -rf build
python setup.py build
python setup.py install
```

La compilación tarda entre 5-10 minutos.

### 7. Verificar

```bash
python -c "import pydnp3; print('DNP3 instalado')"
```

Si sale "DNP3 instalado" sin errores, funcionó.

## Script completo

Si quieres hacerlo todo automático:

```bash
#!/bin/bash
set -e

cd ~/Proyectos/rapido
mkdir -p dnp3
cd dnp3

python -m venv .venv
source .venv/bin/activate
pip install --upgrade pip setuptools wheel

if [ ! -d "dnp3-python" ]; then
    git clone --recursive https://github.com/VOLTTRON/dnp3-python.git
fi
cd dnp3-python

find . \( -name "CMakeLists.txt" -o -name "*.cmake" \) -type f -exec sed -i -E 's/cmake_minimum_required[[:space:]]*\(VERSION[[:space:]]+[0-9]+\.[0-9]+(\.[0-9]+)?\)/cmake_minimum_required(VERSION 3.10)/g' {} \;

cd deps
if [ -d "pybind11" ]; then
    mv pybind11 pybind11.old
fi
git clone https://github.com/pybind/pybind11.git pybind11
cd pybind11
git checkout v2.13.6
cd ../..

rm -rf build
python setup.py build
python setup.py install

python -c "import pydnp3; print('Listo')"
```

## Resumen de cambios

| Archivo | Problema | Solución |
|---------|----------|----------|
| CMakeLists.txt | cmake 2.8 | Cambiar a 3.10 |
| *.cmake | cmake 2.8.12 | Cambiar a 3.10 |
| deps/pybind11 | No compila en Python 3.12 | Usar v2.13.6 |

## Notas

- No usar Python 3.13 (tiene más cambios incompatibles en el C API)
- La compilación necesita unos 2GB de RAM
- En Arch no hay paquetes python3-dev como en Ubuntu

## Si algo falla

### Error de CMake 3.5
Verificar que todos los archivos están parcheados:

```bash
find . -name "CMakeLists.txt" -exec grep -H "cmake_minimum_required" {} \;
find . -name "*.cmake" -exec grep -H "cmake_minimum_required" {} \;
```

### Error de PyFrameObject
Asegurarse de que pybind11 es la versión 2.13.6:

```bash
cd deps/pybind11
git describe --tags
```

### No encuentra pydnp3
Activar el entorno virtual:

```bash
source ~/Proyectos/rapido/dnp3/.venv/bin/activate
```

## Ejemplo básico

```python
from pydnp3 import opendnp3, asiodnp3, asiopal

manager = asiodnp3.DNP3Manager(1)
asiodnp3.ConsoleLogger().Create()

canal = manager.AddTCPClient(
    "cliente",
    opendnp3.levels.NORMAL,
    asiopal.ChannelRetry().Default(),
    "127.0.0.1",
    "0.0.0.0",
    20000,
    asiodnp3.PrintingChannelListener().Create()
)
```

## Referencias

- https://github.com/VOLTTRON/dnp3-python
- https://github.com/pybind/pybind11
- https://github.com/dnp3/opendnp3 (archivado)

---

Testeado en Arch Linux con Python 3.12.11 y CMake 4.0.1
