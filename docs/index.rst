PT Documentation
================

.. image:: _static/pt.svg
   :alt: PT Logo
   :width: 100
   :align: center

**PT** is a powerful CLI tool that writes your clipboard content directly to a file – with automatic timestamped backups, backup comments, recursive file search, delta diff comparison, directory tree visualization, and safe file deletion. It's not just a clipboard manager – it's a complete version control system for your files!

.. -- toctree hidden is used to include all docs in sidebar
.. toctree::
   :maxdepth: 3
   :caption: Contents
   :hidden:

   quickstart
   installation
   usage
   configuration
   features
   workflow
   examples
   troubleshooting
   contributing
   license

Overview
--------

Core Features
~~~~~~~~~~~~~

- 📝 **Quick Save** - Write clipboard content to file with one command
- 📦 **Auto Backup** - Automatic timestamped backups stored in `.pt/` directory
- 💬 **Backup Comments** - Add descriptive comments to track why changes were made
- 🔄 **Restore** - Interactive or quick restore from backups with comments
- 📊 **Beautiful Listings** - Formatted table view of all backups with sizes and comments
- 🔍 **Recursive File Search** - Automatically finds files in subdirectories up to 10 levels deep
- 📊 **Delta Diff Integration** - Beautiful side-by-side diff comparison with backups
- 🌳 **Directory Tree View** - Visual file structure with sizes (like `tree` command)
- 🗑️ **Safe Delete** - Backup before deletion, create empty placeholder
- ⚙️ **Configurable** - Customize behavior via `pt.yml` config file

Git-Like Workflow (NEW!)
~~~~~~~~~~~~~~~~~~~~~~~~~

PT introduces a powerful git-like workflow for version management:

- ``pt check`` - Show status of all files (like ``git status``)
- ``pt commit -m "message"`` - Backup all changed files (like ``git commit``)
- ``pt -l <file>`` - View commit history
- ``pt -d <file> --last`` - See what changed (diff)
- ``pt -r <file> --last`` - Rollback if needed

Requirements
------------

- Go 1.16 or higher
- Git (for installation)
- **Delta** (optional, for diff functionality)

Quick Start
-----------

Copy text to clipboard, then::

   pt notes.txt -m "Initial version"

For more details, see the :doc:`quickstart` guide.

Indices and Tables
==================

* :ref:`genindex`
* :ref:`modindex`
* :ref:`search`