#!/usr/bin/env python3
"""Generate Mermaid ER diagrams from Spec Kit data-model.md files.

Authored by: OpenCode
"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path


COUNT_TOKEN_MAP = {
    "zero or one": "o|",
    "zero or more": "o{",
    "one or more": "|{",
    "exactly one": "||",
    "many": "o{",
    "one": "||",
}

COUNTED_ENTITY_PATTERN = re.compile(
    r"(?P<count>zero or one|zero or more|one or more|exactly one|many|one)"
    r"\s+(?:new\s+)?`(?P<entity>[^`]+)`",
    re.IGNORECASE,
)
ENTITY_HEADING_PATTERN = re.compile(r"^##\s+(.+?)\s*$")
FIELDS_SECTION_PATTERN = re.compile(r"^#{0,6}\s*Fields:?\s*$", re.IGNORECASE)
RELATIONSHIPS_SECTION_PATTERN = re.compile(r"^#{0,6}\s*Relationships:?\s*$", re.IGNORECASE)

RELATIONSHIP_LABEL_PATTERNS = (
    (re.compile(r"^wraps\b", re.IGNORECASE), "wraps"),
    (re.compile(r"^contains\b", re.IGNORECASE), "contains"),
    (re.compile(r"^owns\b", re.IGNORECASE), "owns"),
    (re.compile(r"^uses\b", re.IGNORECASE), "uses"),
    (re.compile(r"^belongs to\b", re.IGNORECASE), "belongs_to"),
    (re.compile(r"^consumes\b", re.IGNORECASE), "consumes"),
    (re.compile(r"^produces\b", re.IGNORECASE), "produces"),
    (re.compile(r"^derived from\b", re.IGNORECASE), "derived_from"),
    (re.compile(r"^optionally references\b", re.IGNORECASE), "references"),
    (re.compile(r"^references\b", re.IGNORECASE), "references"),
    (re.compile(r"^may be referenced by\b", re.IGNORECASE), "referenced_by"),
    (re.compile(r"^is referenced by\b", re.IGNORECASE), "referenced_by"),
)


@dataclass(frozen=True)
class Field:
    """Store one field row extracted from a markdown table."""

    name: str
    raw_type: str
    notes: str = ""


@dataclass
class Entity:
    """Store one entity section extracted from a Spec Kit data model."""

    name: str
    fields: list[Field] = field(default_factory=list)
    relationship_lines: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class Relationship:
    """Store one Mermaid relationship between two parsed entities."""

    source: str
    target: str
    source_cardinality: str
    target_cardinality: str
    label: str


def strip_code_ticks(value: str) -> str:
    """Remove surrounding markdown code ticks from a value."""

    return value.strip().strip("`").strip()


def sanitize_identifier(value: str) -> str:
    """Convert a name into a Mermaid-safe identifier."""

    identifier = re.sub(r"[^A-Za-z0-9_]", "_", strip_code_ticks(value))
    identifier = re.sub(r"_+", "_", identifier).strip("_")

    if not identifier:
        return "UNNAMED"

    if identifier[0].isdigit():
        return f"ID_{identifier}"

    return identifier


def split_table_row(line: str) -> list[str] | None:
    """Split a markdown table row into stripped cells."""

    stripped = line.strip()
    if not stripped.startswith("|"):
        return None

    return [cell.strip() for cell in stripped.strip("|").split("|")]


def is_separator_row(cells: list[str]) -> bool:
    """Report whether a markdown table row is the separator line."""

    return bool(cells) and all(re.fullmatch(r":?-{3,}:?", cell) for cell in cells)


def cell_at(cells: list[str], index: int | None) -> str:
    """Return a cell value when the requested column exists."""

    if index is None or index >= len(cells):
        return ""

    return cells[index]


def parse_fields_table(lines: list[str], start_index: int) -> tuple[list[Field], int]:
    """Parse a markdown fields table that starts after a Fields section marker."""

    index = start_index
    while index < len(lines) and not lines[index].strip():
        index += 1

    header = split_table_row(lines[index]) if index < len(lines) else None
    separator = split_table_row(lines[index + 1]) if index + 1 < len(lines) else None
    if not header or not separator or not is_separator_row(separator):
        return [], start_index

    normalized_headers = [cell.lower() for cell in header]
    field_index = next((i for i, cell in enumerate(normalized_headers) if cell in {"field", "name", "property"}), None)
    type_index = next((i for i, cell in enumerate(normalized_headers) if cell == "type"), None)
    notes_index = next((i for i, cell in enumerate(normalized_headers) if cell in {"notes", "description", "details"}), None)

    if field_index is None or type_index is None:
        return [], start_index

    fields: list[Field] = []
    index += 2
    while index < len(lines):
        cells = split_table_row(lines[index])
        if not cells:
            break
        if is_separator_row(cells):
            index += 1
            continue

        name = strip_code_ticks(cell_at(cells, field_index))
        raw_type = strip_code_ticks(cell_at(cells, type_index))
        notes = strip_code_ticks(cell_at(cells, notes_index))
        if name and raw_type:
            fields.append(Field(name=name, raw_type=raw_type, notes=notes))
        index += 1

    return fields, index


def parse_bullet_list(lines: list[str], start_index: int) -> tuple[list[str], int]:
    """Parse a markdown bullet list, including indented continuation lines."""

    items: list[str] = []
    parts: list[str] = []
    index = start_index

    while index < len(lines) and not lines[index].strip():
        index += 1

    while index < len(lines):
        stripped = lines[index].strip()
        if stripped.startswith("- ") or stripped.startswith("* "):
            if parts:
                items.append(" ".join(parts))
            parts = [stripped[2:].strip()]
            index += 1
            continue

        if parts and stripped and (lines[index].startswith("  ") or lines[index].startswith("\t")):
            parts.append(stripped)
            index += 1
            continue

        break

    if parts:
        items.append(" ".join(parts))

    return items, index


def parse_entities(markdown: str) -> list[Entity]:
    """Extract entity sections, fields, and relationship bullets from markdown."""

    lines = markdown.splitlines()
    entities: list[Entity] = []
    current: Entity | None = None
    index = 0

    while index < len(lines):
        heading_match = ENTITY_HEADING_PATTERN.match(lines[index].strip())
        if heading_match:
            current = Entity(name=strip_code_ticks(heading_match.group(1)))
            entities.append(current)
            index += 1
            continue

        if current is None:
            index += 1
            continue

        stripped = lines[index].strip()
        if FIELDS_SECTION_PATTERN.match(stripped):
            fields, index = parse_fields_table(lines, index + 1)
            current.fields.extend(fields)
            continue

        if RELATIONSHIPS_SECTION_PATTERN.match(stripped):
            relationships, index = parse_bullet_list(lines, index + 1)
            current.relationship_lines.extend(relationships)
            continue

        index += 1

    return [entity for entity in entities if entity.fields or entity.relationship_lines]


def relationship_key(source: str, target: str) -> tuple[str, str]:
    """Build a stable deduplication key for an entity pair."""

    return tuple(sorted((source, target)))


def relationship_label(text: str) -> str:
    """Normalize a relationship sentence into a Mermaid label token."""

    for pattern, label in RELATIONSHIP_LABEL_PATTERNS:
        if pattern.search(text):
            return label

    prefix = COUNTED_ENTITY_PATTERN.split(text, maxsplit=1)[0]
    prefix = re.sub(r"[^A-Za-z0-9]+", "_", prefix.lower()).strip("_")
    return prefix or "relates_to"


def relationship_cardinality(text: str, count: str) -> str:
    """Map a relationship sentence and count token to a Mermaid cardinality."""

    normalized_count = count.lower()
    if normalized_count == "one" and re.search(r"\b(optionally|optional|may)\b", text, re.IGNORECASE):
        return "o|"

    return COUNT_TOKEN_MAP[normalized_count]


def is_collection_type(raw_type: str) -> bool:
    """Report whether a field type represents a collection."""

    lower = strip_code_ticks(raw_type).lower()
    return "[]" in raw_type or bool(re.search(r"\b(array|list|set|collection)\b", lower))


def is_nullable(raw_type: str, notes: str) -> bool:
    """Report whether a field is explicitly nullable or optional."""

    combined = f"{raw_type} {notes}".lower()
    return bool(re.search(r"\b(nullable|optional|null|nil)\b", combined))


def entity_reference(raw_type: str, entity_names: list[str]) -> str | None:
    """Find an entity name mentioned inside a raw field type."""

    cleaned = strip_code_ticks(raw_type)
    for entity_name in sorted(entity_names, key=len, reverse=True):
        pattern = re.compile(rf"(?<![A-Za-z0-9_]){re.escape(entity_name)}(?![A-Za-z0-9_])")
        if pattern.search(cleaned):
            return entity_name

    return None


def mermaid_type(raw_type: str, entity_names: list[str]) -> str:
    """Map a raw markdown field type to a generic Mermaid ER type."""

    lower = strip_code_ticks(raw_type).lower()
    if "uuid" in lower:
        return "UUID"
    if is_collection_type(raw_type):
        return "ARRAY"
    if re.search(r"\b(decimal|numeric|number|float|double)\b", lower):
        return "DECIMAL"
    if re.search(r"\b(integer|int)\b", lower):
        return "INTEGER"
    if re.search(r"\b(boolean|bool)\b", lower):
        return "BOOLEAN"
    if re.search(r"\b(timestamp|datetime|date|time)\b", lower):
        return "TIMESTAMP"
    if "enum" in lower:
        return "ENUM"
    if re.search(r"\b(bytes|byte|binary|blob)\b", lower):
        return "BYTES"
    if re.search(r"\b(string|text|char|url|uri|email)\b", lower):
        return "STRING"
    if entity_reference(raw_type, entity_names) or re.search(r"\b(object|map|json|record)\b", lower):
        return "OBJECT"

    return "STRING"


def explicit_relationships(entities: list[Entity]) -> tuple[list[Relationship], set[tuple[str, str]]]:
    """Extract explicit relationships from relationship bullet lists."""

    entity_names = [entity.name for entity in entities]
    relationships: list[Relationship] = []
    seen_pairs: set[tuple[str, str]] = set()

    for entity in entities:
        for line in entity.relationship_lines:
            label = relationship_label(line)
            for match in COUNTED_ENTITY_PATTERN.finditer(line):
                target = strip_code_ticks(match.group("entity"))
                if target not in entity_names or target == entity.name:
                    continue

                key = relationship_key(entity.name, target)
                if key in seen_pairs:
                    continue

                relationships.append(
                    Relationship(
                        source=entity.name,
                        target=target,
                        source_cardinality="||",
                        target_cardinality=relationship_cardinality(line, match.group("count")),
                        label=label,
                    )
                )
                seen_pairs.add(key)

    return relationships, seen_pairs


def inferred_relationships(entities: list[Entity], seen_pairs: set[tuple[str, str]]) -> list[Relationship]:
    """Infer missing relationships from entity-typed fields."""

    entity_names = [entity.name for entity in entities]
    relationships: list[Relationship] = []

    for entity in entities:
        for field_value in entity.fields:
            target = entity_reference(field_value.raw_type, entity_names)
            if target is None or target == entity.name:
                continue

            key = relationship_key(entity.name, target)
            if key in seen_pairs:
                continue

            target_cardinality = "o{" if is_collection_type(field_value.raw_type) else "o|" if is_nullable(field_value.raw_type, field_value.notes) else "||"
            label = "contains" if is_collection_type(field_value.raw_type) else "references"

            relationships.append(
                Relationship(
                    source=entity.name,
                    target=target,
                    source_cardinality="||",
                    target_cardinality=target_cardinality,
                    label=label,
                )
            )
            seen_pairs.add(key)

    return relationships


def render_mermaid(entities: list[Entity], relationships: list[Relationship]) -> str:
    """Render the parsed entities and relationships as Mermaid ER source."""

    entity_names = [entity.name for entity in entities]
    lines = ["erDiagram"]

    for entity in entities:
        lines.append(f"    {sanitize_identifier(entity.name)} {{")
        for field_value in entity.fields:
            lines.append(
                f"        {mermaid_type(field_value.raw_type, entity_names)} {sanitize_identifier(field_value.name)}"
            )
        lines.append("    }")
        lines.append("")

    for relationship in relationships:
        lines.append(
            "    "
            f"{sanitize_identifier(relationship.source)} {relationship.source_cardinality}--{relationship.target_cardinality} "
            f"{sanitize_identifier(relationship.target)} : {relationship.label}"
        )

    while lines and lines[-1] == "":
        lines.pop()

    return "\n".join(lines)


def parse_arguments() -> argparse.Namespace:
    """Parse command-line arguments for the generator."""

    parser = argparse.ArgumentParser(
        description="Generate data-model-diagram.mmd from a Spec Kit data-model.md file."
    )
    parser.add_argument("input_path", type=Path)
    parser.add_argument("output_path", type=Path)
    return parser.parse_args()


def main() -> int:
    """Load a markdown data model and write the Mermaid diagram output."""

    args = parse_arguments()

    try:
        markdown = args.input_path.read_text(encoding="utf-8")
        entities = parse_entities(markdown)
        if not entities:
            raise ValueError(f"no entities were parsed from {args.input_path}")

        explicit, seen_pairs = explicit_relationships(entities)
        inferred = inferred_relationships(entities, seen_pairs)
        output = render_mermaid(entities, explicit + inferred)

        args.output_path.parent.mkdir(parents=True, exist_ok=True)
        args.output_path.write_text(f"{output}\n", encoding="utf-8")
    except Exception as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
