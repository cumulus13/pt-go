.. _installation:

Installation
============

Prerequisites
-------------

* **Go Compiler**: If building from source, ensure you have Go installed (e.g., Go 1.19 or later).
* **Delta (Optional)**: For ``pt -d`` (diff) functionality, install `delta <https://github.com/dandavison/delta>`_.

Building from Source
--------------------

1. Clone or download the source code containing ``pt/main.go``.
2. Navigate to the directory containing ``pt/main.go``.
3. Run the following command to build the executable:

   * On Linux/macOS:

     .. code-block:: bash

        go build -o pt main.go

   * On Windows:

     .. code-block:: batch

        go build -o pt.exe main.go

4. The resulting ``pt`` (Linux/macOS) or ``pt.exe`` (Windows) executable can be run directly or placed in your system's PATH.

Using a Pre-built Binary
------------------------

If a pre-built binary is available, download it for your operating system and place it in your PATH.