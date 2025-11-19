# Configuration file for the Sphinx documentation builder.
import os
import sys

# -- Project information -----------------------------------------------------
project = 'PT - Clipboard to File Tool with Smart Version Management'
copyright = '2025, Hadi Cahyadi'
author = 'Hadi Cahyadi'
release = '1.0.25'

# -- General configuration ---------------------------------------------------
extensions = [
    'sphinx.ext.autodoc',
    'sphinx.ext.viewcode',
    'sphinx.ext.napoleon',
    'sphinx.ext.intersphinx',
    'sphinx.ext.todo',
    'sphinx.ext.coverage',
    'sphinx.ext.mathjax',
    'sphinx.ext.ifconfig',
    'sphinx.ext.githubpages',
    # 'sphinxarg.ext',  # Removed
    'sphinx_material',  # Tambahkan tema sphinx_material
]

templates_path = ['_templates']
exclude_patterns = ['_build', 'Thumbs.db', '.DS_Store']

language = 'en'

# -- Options for HTML output -------------------------------------------------
html_theme = 'sphinx_material'
html_theme_options = {
    # Global
    'nav_title': 'PT - Clipboard to File Tool with Smart Version Management',
    'globaltoc_depth': 2,
    'globaltoc_collapse': True,
    'globaltoc_includehidden': True,
    'master_doc': False,

    # Color scheme: rxvt-inspired dark terminal
    'color_primary': 'black',        # Background
    'color_accent': 'green',         # Primary accent (like rxvt green text)
    # 'theme_color': '#000000',     # Ini bukan opsi yang didukung secara langsung
    # 'palette': [
    #     {
    #         'media': '(prefers-color-scheme: light)',
    #         'scheme': 'default',
    #         'primary': 'blue',
    #         'accent': 'light-blue',
    #         'toggle': {
    #             'icon': 'material/lightbulb-outline',
    #             'name': 'Switch to dark mode',
    #         },
    #     },
    #     {
    #         'media': '(prefers-color-scheme: dark)',
    #         'scheme': 'slate',
    #         'primary': 'black',
    #         'accent': 'green',
    #         'toggle': {
    #             'icon': 'material/lightbulb',
    #             'name': 'Switch to light mode',
    #         },
    #     },
    # ],

    # Font
    # 'font': {                      # Opsi ini mungkin tidak didukung secara penuh
    #     'text': 'monospace',
    #     'code': 'monospace',
    # },
    # 'css_url': '_static/custom.css',  # Diganti dengan html_css_files

    # Logo
    # 'logo_icon': 'fa fa-terminal',
}

html_logo = '_static/pt.svg'

# Custom CSS for terminal-like styling
html_static_path = ['_static']

# Add custom CSS file to override default styles
html_css_files = [
    'custom.css',
    'https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css',
]

# Use monospace font for code blocks and inline code (via CSS)
# html_context = { ... } # Removed, as it's not needed for CSS inclusion

# -- Options for LaTeX output ------------------------------------------------
latex_elements = {}

# -- Options for todo extension ----------------------------------------------
todo_include_todos = True

rst_prolog = """
.. |project_name| replace:: PT - Clipboard to File Tool with Smart Version Management
.. |author| replace:: Hadi Cahyadi
.. |version| replace:: 1.0.25
"""