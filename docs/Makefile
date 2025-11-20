# Minimal makefile for Sphinx documentation
#

# You can set these variables from the command line, and also
# from the environment for the first two.
SPHINXOPTS    ?=
SPHINXBUILD  ?= sphinx-build
SOURCEDIR    = .
BUILDDIR     = _build

# Put it first so that "make" without argument is like "make help".
help:
	@$(SPHINXBUILD) -M help "$(SOURCEDIR)" "$(BUILDDIR)" $(SPHINXOPTS) $(O)

.PHONY: help Makefile

# Catch-all target: route all unknown targets to Sphinx
# Use `make mode='view'` to open the generated docs in a browser after building.
%: Makefile
	@$(SPHINXBUILD) -M $@ "$(SOURCEDIR)" "$(BUILDDIR)" $(SPHINXOPTS) $(O)
	@if [ "$@" = "html" ] && [ "$(mode)" = "view" ]; then \
		echo "Opening docs in browser..."; \
		python -m http.server --directory "$(BUILDDIR)/html" 8000 & \
		sleep 2; \
		python -c "import webbrowser; webbrowser.open('http://localhost:8000')"; \
	fi