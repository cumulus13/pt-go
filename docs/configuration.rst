.. _configuration:

Configuration
=============

|project_name| can be configured using a YAML file named ``pt.yml`` (or ``pt.yaml``, ``.pt.yml``, ``.pt.yaml``). It searches for this file in the current directory, ``~/.config/pt/``, or ``~/.pt/``.

Default Configuration
---------------------

The default settings are:

.. code-block:: yaml

   # PT Configuration File
   # This file configures the behavior of the PT tool
   # All values are optional - if not specified, defaults will be used
   # Maximum clipboard content size in bytes (default: 104857600 = 100MB)
   # Range: 1 - 1073741824 (1GB)
   max_clipboard_size: 104857600
   max_backup_count: 100
   max_filename_length: 200
   backup_dir_name: ".pt"
   max_search_depth: 10

Configuration Options
---------------------

* **max_clipboard_size** (int): Maximum allowed size of clipboard content in bytes. Default: 104857600 (100MB).
* **max_backup_count** (int): Maximum number of backups to keep per file. Default: 100.
* **max_filename_length** (int): Maximum allowed length for filenames. Default: 200.
* **backup_dir_name** (string): Name of the hidden directory used for backups. Default: ".pt".
* **max_search_depth** (int): Maximum directory depth for recursive file searches. Default: 10.

Creating a Configuration File
-----------------------------

Use the following command to generate a sample configuration file named ``pt.yml`` in the current directory:

.. code-block:: bash

   pt config init

You can then edit this file to change the default settings.