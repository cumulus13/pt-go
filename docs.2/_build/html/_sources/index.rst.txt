.. PT - Clipboard to File Tool with Smart Version Management documentation master file, created by
   sphinx-quickstart on Wed Nov 19 2025.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

Welcome to PT - Clipboard to File Tool with Smart Version Management's documentation!
======================================================================================


.. image:: _static/pt.svg  
   :alt: Pt icon
   :align: center
   :width: 320

.. _introduction:

Introduction
============

|project_name| is a production-hardened command-line tool designed to seamlessly transfer content from your clipboard into files. It functions similarly to Git, using a hidden ``.pt`` directory to store backups of file changes, enabling version control-like operations for clipboard content.

Key Features
------------

* **Terminal-Style Workflow**: Uses a ``.pt`` directory structure for backups, mirroring Git's approach.
* **Clipboard Integration**: Reads content directly from the system clipboard.
* **Backup Management**: Automatically creates timestamped backups before overwriting files.
* **Status Checking**: Compare current files against their last backed-up versions (like ``git status``).
* **Commit Changes**: Back up all modified and new files in the current directory tree (like ``git commit``).
* **Recursive Search**: Finds files by name in subdirectories, respecting ``.gitignore`` and ``.ptignore``.
* **Diff Support**: Compare files with their backups using `delta <https://github.com/dandavison/delta>`_.
* **Safe Operations**: Includes path traversal checks, system directory protection, and size limits.
* **Cross-Platform**: Works on Linux, macOS, and Windows.

License
-------

This project is licensed under the MIT License - see the :ref:`license` page for details.


Indices and tables
==================

.. toctree::
   :maxdepth: 2
   :caption: Contents:

   introduction
   installation
   usage
   commands
   configuration
   api
   contributing
   license

* :ref:`genindex`
* :ref:`modindex`
* :ref:`search`