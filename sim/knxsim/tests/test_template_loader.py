"""Unit tests for TemplateLoader."""

from __future__ import annotations

import os


def test_load_all_templates(template_loader):
    templates = template_loader.list_templates()
    assert len(templates) == 57
    for tpl in templates:
        assert "id" in tpl
        assert "name" in tpl
        assert "domain" in tpl
        assert "group_addresses" in tpl


def test_get_template(template_loader):
    template = template_loader.get_template("light_switch")
    assert template is not None
    assert template.id == "light_switch"

    missing = template_loader.get_template("nonexistent")
    assert missing is None


def test_template_group_addresses_are_dicts(template_loader):
    templates = template_loader.list_templates()
    for tpl in templates:
        for ga in tpl.get("group_addresses", {}).values():
            assert isinstance(ga, dict)
            assert "dpt" in ga


def test_template_domain_matches_directory(template_loader):
    for template in template_loader._templates.values():
        domain_dir = os.path.basename(os.path.dirname(template.file_path))
        assert template.domain == domain_dir
