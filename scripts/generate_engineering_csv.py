#!/usr/bin/env python3
"""Generate related engineering data CSV files for testing pipelines.

This script produces multiple CSV tables with shared keys such as ``part_id``.
The number of generated rows can be controlled via command-line options.

Example usage::

    python scripts/generate_engineering_csv.py \
        --output ./tmp-data \
        --parts 250 \
        --measurements 500 \
        --assemblies 400 \
        --suppliers 25

"""

from __future__ import annotations

import argparse
import csv
import random
from dataclasses import dataclass
from datetime import datetime, timedelta
from pathlib import Path
from typing import Iterable, List, Sequence

MATERIALS = [
    "Aluminum",
    "Steel",
    "Titanium",
    "Carbon Fiber",
    "ABS Plastic",
    "Ceramic",
]

OPERATORS = [
    "Alice",
    "Bob",
    "Charlie",
    "Dina",
    "Evelyn",
    "Frank",
]

ASSEMBLY_STEPS = [
    "Fasten bolts",
    "Apply adhesive",
    "Install wiring",
    "Torque calibration",
    "Visual inspection",
]

MEASUREMENT_TYPES = [
    ("Length", "mm"),
    ("Mass", "kg"),
    ("Temperature", "C"),
    ("Pressure", "kPa"),
    ("Voltage", "V"),
]

SUPPLIERS = [
    "Acme Components",
    "Orbital Supplies",
    "Precision Parts Co.",
    "Rocketry Wholesale",
    "Titan Manufacturing",
]


@dataclass
class Part:
    part_id: str
    part_number: str
    description: str
    weight_kg: float
    material: str
    created_at: datetime

    def to_row(self) -> List[str]:
        return [
            self.part_id,
            self.part_number,
            self.description,
            f"{self.weight_kg:.3f}",
            self.material,
            self.created_at.isoformat(timespec="seconds"),
        ]


@dataclass
class Assembly:
    assembly_id: str
    part_id: str
    assembly_step: str
    torque_nm: float
    operator: str
    timestamp: datetime

    def to_row(self) -> List[str]:
        return [
            self.assembly_id,
            self.part_id,
            self.assembly_step,
            f"{self.torque_nm:.2f}",
            self.operator,
            self.timestamp.isoformat(timespec="seconds"),
        ]


@dataclass
class Measurement:
    measurement_id: str
    part_id: str
    measurement_type: str
    value: float
    units: str
    measured_at: datetime

    def to_row(self) -> List[str]:
        return [
            self.measurement_id,
            self.part_id,
            self.measurement_type,
            f"{self.value:.3f}",
            self.units,
            self.measured_at.isoformat(timespec="seconds"),
        ]


@dataclass
class SupplierRelationship:
    supplier_part_id: str
    part_id: str
    supplier_name: str
    cost_usd: float
    lead_time_days: int
    qualified_on: datetime

    def to_row(self) -> List[str]:
        return [
            self.supplier_part_id,
            self.part_id,
            self.supplier_name,
            f"{self.cost_usd:.2f}",
            str(self.lead_time_days),
            self.qualified_on.isoformat(timespec="seconds"),
        ]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--output",
        type=Path,
        default=Path("./engineering-data"),
        help="Directory to write CSV files into (created if missing).",
    )
    parser.add_argument(
        "--seed",
        type=int,
        default=None,
        help="Optional seed for the random number generator.",
    )
    parser.add_argument("--parts", type=int, default=100, help="Number of parts to generate.")
    parser.add_argument(
        "--assemblies",
        type=int,
        default=200,
        help="Number of assembly log entries to generate.",
    )
    parser.add_argument(
        "--measurements",
        type=int,
        default=300,
        help="Number of measurement records to generate.",
    )
    parser.add_argument(
        "--suppliers",
        type=int,
        default=50,
        help="Number of supplier-part relationships to generate.",
    )
    return parser.parse_args()


def ensure_output_dir(path: Path) -> None:
    path.mkdir(parents=True, exist_ok=True)


def choice(sequence: Sequence[str]) -> str:
    return random.choice(sequence)


def generate_parts(count: int) -> List[Part]:
    parts: List[Part] = []
    base_date = datetime.now() - timedelta(days=365)
    for idx in range(1, count + 1):
        part_id = f"PART-{idx:06d}"
        part_number = f"PN-{random.randint(10_000, 99_999)}"
        description = f"Component {idx}"
        weight_kg = random.uniform(0.1, 75.0)
        material = choice(MATERIALS)
        created_at = base_date + timedelta(days=random.randint(0, 365))
        parts.append(
            Part(
                part_id=part_id,
                part_number=part_number,
                description=description,
                weight_kg=weight_kg,
                material=material,
                created_at=created_at,
            )
        )
    return parts


def generate_assemblies(parts: Sequence[Part], count: int) -> List[Assembly]:
    assemblies: List[Assembly] = []
    for idx in range(1, count + 1):
        part = random.choice(parts)
        assembly_step = choice(ASSEMBLY_STEPS)
        torque_nm = random.uniform(1.0, 150.0)
        operator = choice(OPERATORS)
        timestamp = part.created_at + timedelta(hours=random.randint(1, 500))
        assemblies.append(
            Assembly(
                assembly_id=f"ASM-{idx:06d}",
                part_id=part.part_id,
                assembly_step=assembly_step,
                torque_nm=torque_nm,
                operator=operator,
                timestamp=timestamp,
            )
        )
    return assemblies


def generate_measurements(parts: Sequence[Part], count: int) -> List[Measurement]:
    measurements: List[Measurement] = []
    for idx in range(1, count + 1):
        part = random.choice(parts)
        measurement_type, units = choice(MEASUREMENT_TYPES)
        value = random.uniform(0.0, 1000.0)
        measured_at = part.created_at + timedelta(hours=random.randint(2, 720))
        measurements.append(
            Measurement(
                measurement_id=f"MEAS-{idx:06d}",
                part_id=part.part_id,
                measurement_type=measurement_type,
                value=value,
                units=units,
                measured_at=measured_at,
            )
        )
    return measurements


def generate_supplier_relationships(parts: Sequence[Part], count: int) -> List[SupplierRelationship]:
    relationships: List[SupplierRelationship] = []
    for idx in range(1, count + 1):
        part = random.choice(parts)
        supplier_name = choice(SUPPLIERS)
        cost_usd = random.uniform(25.0, 10_000.0)
        lead_time_days = random.randint(1, 120)
        qualified_on = part.created_at - timedelta(days=random.randint(0, 60))
        relationships.append(
            SupplierRelationship(
                supplier_part_id=f"SUP-{idx:06d}",
                part_id=part.part_id,
                supplier_name=supplier_name,
                cost_usd=cost_usd,
                lead_time_days=lead_time_days,
                qualified_on=qualified_on,
            )
        )
    return relationships


def write_csv(path: Path, headers: Sequence[str], rows: Iterable[Sequence[str]]) -> None:
    with path.open("w", newline="", encoding="utf-8") as csvfile:
        writer = csv.writer(csvfile)
        writer.writerow(headers)
        writer.writerows(rows)


def main() -> None:
    args = parse_args()
    if args.seed is not None:
        random.seed(args.seed)

    ensure_output_dir(args.output)
    parts = generate_parts(args.parts)

    write_csv(
        args.output / "parts.csv",
        ["part_id", "part_number", "description", "weight_kg", "material", "created_at"],
        (part.to_row() for part in parts),
    )

    assemblies = generate_assemblies(parts, args.assemblies)
    write_csv(
        args.output / "assemblies.csv",
        ["assembly_id", "part_id", "assembly_step", "torque_nm", "operator", "timestamp"],
        (assembly.to_row() for assembly in assemblies),
    )

    measurements = generate_measurements(parts, args.measurements)
    write_csv(
        args.output / "measurements.csv",
        ["measurement_id", "part_id", "measurement_type", "value", "units", "measured_at"],
        (measurement.to_row() for measurement in measurements),
    )

    suppliers = generate_supplier_relationships(parts, args.suppliers)
    write_csv(
        args.output / "supplier_relationships.csv",
        ["supplier_part_id", "part_id", "supplier_name", "cost_usd", "lead_time_days", "qualified_on"],
        (supplier.to_row() for supplier in suppliers),
    )

    print(f"Wrote CSV files to {args.output.resolve()}")


if __name__ == "__main__":
    main()
