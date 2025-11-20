import os
import sys
from version_get import VersionGet  
# -- Project information -----------------------------------------------------
project = 'PT'
copyright = '2025, Hadi Cahyadi'
author = 'Hadi Cahyadi'
release = VersionGet().get(True) if not VersionGet().get(True) == "1.0.0" else '1.0.35'
version = VersionGet().get(True) if not VersionGet().get(True) == "1.0.0" else '1.0.35'

# -- General configuration ---------------------------------------------------
sys.path.insert(0, os.path.abspath('..'))

extensions = [
    'sphinx.ext.autodoc',
    'sphinx.ext.viewcode',
    'sphinx.ext.intersphinx',
    'sphinx.ext.extlinks',
    'sphinx_copybutton',
    'sphinx_tabs.tabs',
    'sphinx_toolbox.collapse',
]

templates_path = ['_templates']
exclude_patterns = ['_build', 'Thumbs.db', '.DS_Store']
source_suffix = '.rst'
master_doc = 'index'
language = 'en'

# -- Syntax highlighting -----------------------------------------------------
pygments_style = 'monokai'
pygments_dark_style = 'monokai'

# -- HTML theme --------------------------------------------------------------
html_theme = 'sphinx_rtd_theme'

html_theme_options = {
    'logo_only': True,
    'prev_next_buttons_location': 'bottom',
    'style_external_links': True,
    'vcs_pageview_mode': '',
    'style_nav_header_background': '#0C0C0C',
    'collapse_navigation': False,
    'sticky_navigation': True,
    'navigation_depth': -1,
    'includehidden': True,
    'titles_only': False,
}

html_static_path = ['_static']
html_css_files = ['terminal-dark.css']
html_favicon = '_static/pt.svg'
html_logo = '_static/pt.svg'
html_title = 'PT Documentation'
html_short_title = 'PT'

# -- Extension configuration -------------------------------------------------
copybutton_prompt_text = r'\$ |>>> |\.\.\. '
copybutton_prompt_is_regexp = True
copybutton_line_continuation_character = '\\'
copybutton_here_doc_delimiter = 'EOF'

# -- External links ----------------------------------------------------------
extlinks = {
    'repo': ('https://github.com/cumulus13/pt-go/%s', 'GitHub %s'),
    'issue': ('https://github.com/cumulus13/pt-go/issues/%s', 'Issue #%s'),
}

# -- Intersphinx mapping (disabled for Go due to 404) -----------------------
intersphinx_mapping = {
    'python': ('https://docs.python.org/3', None),
    # 'go': ('https://pkg.go.dev', None),  # Commented out - inventory not available
}

# -- Disable certain warnings ----------------------------------------------
suppress_warnings = [
    'epub.unknown_project_files',
]
