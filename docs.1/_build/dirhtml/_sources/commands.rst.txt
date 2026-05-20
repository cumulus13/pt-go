.. _commands:

Commands
========

This section details the specific commands available in |project_name|.

Write Command
-------------

.. program:: pt

.. option:: <filename>

   The target file to write clipboard content to.

.. option:: -m <msg>, --message <msg>

   Adds a comment to the backup created when the file is overwritten.

.. option:: -c, --check

   Only writes the file if its content differs from the clipboard content.

Append Command
--------------

.. program:: pt

.. option:: + <filename>

   Appends clipboard content to the target file.

Status Command
--------------

.. program:: pt

.. option:: check [filename]

   Shows the status of files. If ``filename`` is provided, shows the status of that specific file.

Commit Command
--------------

.. program:: pt

.. option:: commit -m <message>

   Backs up all files that have been modified or are new since the last backup.

List Command
------------

.. program:: pt

.. option:: -l <filename>, --list <filename>

   Lists all available backups for the specified file.

Restore Command
---------------

.. program:: pt

.. option:: -r <filename>, --restore <filename> [options]

   Restores a backup for the specified file. Use ``--last`` to restore the most recent backup.

Remove Command
--------------

.. program:: pt

.. option:: -rm <filename>, --remove <filename> [options]

   Safely deletes a file by first creating a backup.

Diff Command
------------

.. program:: pt

.. option:: -d <filename>, --diff <filename> [options]

   Compares the current file with a backup. Requires `delta <https://github.com/dandavison/delta>`_.

Tree Command
------------

.. program:: pt

.. option:: -t [path], --tree [path]

   Displays a tree view of the directory structure, respecting ignore files.

Config Command
--------------

.. program:: pt

.. option:: config init [path]

   Creates a sample configuration file.

.. option:: config show

   Shows the current configuration settings.

.. option:: config path

   Shows the location of the configuration file.