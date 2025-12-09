# Configuration file for the Sphinx documentation builder.
# Synced from jbcom-control-center - customize as needed

import os
import sys

# This is a Go project - no Python source path needed

# -- Project information -----------------------------------------------------
project = "secretsync"
copyright = "2025, Jon Bogaty"
author = "Jon Bogaty"

# Try to get version from pyproject.toml or package.json
try:
    import tomllib
    with open("../pyproject.toml", "rb") as f:
        data = tomllib.load(f)
        release = data.get("project", {}).get("version", "0.0.0")
except Exception:
    try:
        import json
        with open("../package.json") as f:
            release = json.load(f).get("version", "0.0.0")
    except Exception:
        release = "0.0.0"

# -- General configuration ---------------------------------------------------

extensions = [
    # Markdown support for Go project documentation
    "myst_parser",
    # Diagrams (optional - requires sphinxcontrib-mermaid)
    # "sphinxcontrib.mermaid",
]

templates_path = ["_templates"]
exclude_patterns = ["_build", "Thumbs.db", ".DS_Store"]

# Source file suffixes
source_suffix = {
    ".rst": "restructuredtext",
    ".md": "markdown",
}

# -- Options for HTML output -------------------------------------------------

html_theme = "sphinx_rtd_theme"
html_static_path = ["_static"]
html_title = f"{project} Documentation"

html_theme_options = {
    "navigation_depth": 4,
    "collapse_navigation": False,
    "sticky_navigation": True,
    "includehidden": True,
    "titles_only": False,
}

# -- Extension configuration -------------------------------------------------

# myst_parser settings
myst_enable_extensions = [
    "colon_fence",
    "deflist",
    "fieldlist",
    "tasklist",
]
myst_heading_anchors = 3
