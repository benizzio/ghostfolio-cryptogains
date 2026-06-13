"""Execute pinned rotki cost-basis code against one generated oracle input.

The adapter imports the verified rotki source tree directly from the untracked
cache path. It does not require a developer-global rotki installation.

Authored by: OpenCode
"""

from __future__ import annotations

import argparse
import enum
import importlib.util
import json
import sys
import types
from dataclasses import dataclass
from pathlib import Path


def main() -> int:
    """Run the local rotki adapter for one generated oracle input.

    Authored by: OpenCode
    """
    parser = argparse.ArgumentParser(prog="rotki_adapter")
    parser.add_argument("--source-root", required=True)
    parser.add_argument("--input", required=True)
    parser.add_argument("--rotki-method", required=True)
    args = parser.parse_args()

    source_root = Path(args.source_root).resolve()
    input_path = Path(args.input).resolve()
    with input_path.open(encoding="utf-8") as handle:
        payload = json.load(handle)

    rotki = load_rotki_boundary(source_root)
    output = execute_rotki_boundary(rotki, payload, args.rotki_method)
    json.dump(output, sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 0


def load_rotki_boundary(source_root: Path) -> dict[str, object]:
    """Load the required rotki modules with local stubs only.

    Authored by: OpenCode
    """
    errors_serialization = types.ModuleType("rotkehlchen.errors.serialization")

    class ConversionError(Exception):
        """Stub conversion error required by rotki's FVal module.

        Authored by: OpenCode
        """

    class DeserializationError(Exception):
        """Stub deserialization error required by rotki modules.

        Authored by: OpenCode
        """

    errors_serialization.ConversionError = ConversionError
    errors_serialization.DeserializationError = DeserializationError
    sys.modules["rotkehlchen.errors.serialization"] = errors_serialization

    fval_module = load_source_module("rotkehlchen.fval", source_root / "rotkehlchen" / "fval.py")
    fval_class = fval_module.FVal

    install_support_stubs(fval_class)
    base_module = load_source_module(
        "rotki_cost_basis_base",
        source_root / "rotkehlchen" / "accounting" / "cost_basis" / "base.py",
    )

    return {
        "Asset": sys.modules["rotkehlchen.assets.asset"].Asset,
        "CostBasisMethod": sys.modules["rotkehlchen.types"].CostBasisMethod,
        "CostBasisEvents": base_module.CostBasisEvents,
        "DBSettings": sys.modules["rotkehlchen.db.settings"].DBSettings,
        "FVal": fval_class,
        "Timestamp": sys.modules["rotkehlchen.types"].Timestamp,
        "AssetAcquisitionEvent": base_module.AssetAcquisitionEvent,
    }


def load_source_module(module_name: str, path: Path) -> types.ModuleType:
    """Load one Python module directly from the verified source tree.

    Authored by: OpenCode
    """
    spec = importlib.util.spec_from_file_location(module_name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"Unable to load module {module_name} from {path}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[module_name] = module
    spec.loader.exec_module(module)
    return module


def install_support_stubs(fval_class: type) -> None:
    """Install the small set of support stubs the rotki cost-basis module needs.

    Authored by: OpenCode
    """
    install_module("rotkehlchen.accounting.types", build_accounting_types_module())
    install_module("rotkehlchen.assets.asset", build_asset_module())
    install_module("rotkehlchen.assets.resolver", build_asset_resolver_module())
    install_module("rotkehlchen.constants", build_constants_module(fval_class))
    install_module("rotkehlchen.constants.assets", build_constants_assets_module())
    install_module("rotkehlchen.db.settings", build_db_settings_module())
    install_module("rotkehlchen.errors.misc", build_accounting_errors_module())
    install_module("rotkehlchen.logging", build_logging_module())
    install_module("rotkehlchen.serialization.deserialize", build_deserialize_module(fval_class))
    install_module("rotkehlchen.types", build_types_module(fval_class))
    install_module("rotkehlchen.user_messages", build_user_messages_module())
    install_module("rotkehlchen.utils.mixins.customizable_date", build_customizable_date_module())


def install_module(name: str, module: types.ModuleType) -> None:
    """Install one in-memory stub module.

    Authored by: OpenCode
    """
    sys.modules[name] = module


def build_accounting_types_module() -> types.ModuleType:
    """Build the minimal `rotkehlchen.accounting.types` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.accounting.types")

    @dataclass
    class MissingAcquisition:
        """Stub missing-acquisition container used by the cost-basis module.

        Authored by: OpenCode
        """

        originating_event_id: int | None
        asset: object
        time: int
        found_amount: object
        missing_amount: object

    class MissingPrice:  # noqa: D401
        """Stub missing-price type.

        Authored by: OpenCode
        """

    module.MissingAcquisition = MissingAcquisition
    module.MissingPrice = MissingPrice
    return module


def build_asset_module() -> types.ModuleType:
    """Build the minimal `rotkehlchen.assets.asset` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.assets.asset")

    class Asset:
        """Stub asset bucket key used by the cost-basis module.

        Authored by: OpenCode
        """

        def __init__(self, identifier: str = "asset") -> None:
            self.identifier = identifier

        def __str__(self) -> str:
            return self.identifier

        def __repr__(self) -> str:
            return self.identifier

        def __hash__(self) -> int:
            return hash(self.identifier)

        def __eq__(self, other: object) -> bool:
            return isinstance(other, Asset) and self.identifier == other.identifier

        def is_fiat(self) -> bool:
            return False

    module.Asset = Asset
    return module


def build_asset_resolver_module() -> types.ModuleType:
    """Build the minimal `rotkehlchen.assets.resolver` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.assets.resolver")

    class AssetResolver:
        """Stub collection resolver used by the cost-basis module.

        Authored by: OpenCode
        """

        @staticmethod
        def get_collection_main_asset(asset_id: str) -> None:
            return None

    module.AssetResolver = AssetResolver
    return module


def build_constants_module(fval_class: type) -> types.ModuleType:
    """Build the minimal `rotkehlchen.constants` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.constants")
    module.ZERO = fval_class(0)
    return module


def build_constants_assets_module() -> types.ModuleType:
    """Build the minimal `rotkehlchen.constants.assets` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.constants.assets")
    asset_class = build_asset_module().Asset
    module.A_ETH = asset_class("ETH")
    module.A_WETH = asset_class("WETH")
    return module


def build_db_settings_module() -> types.ModuleType:
    """Build the minimal `rotkehlchen.db.settings` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.db.settings")

    class DBSettings:
        """Stub DB settings container used by the cost-basis module.

        Authored by: OpenCode
        """

        def __init__(
            self,
            taxfree_after_period: int | None = None,
            main_currency: str = "USD",
            use_asset_collections_in_cost_basis: bool = False,
            cost_basis_method: object | None = None,
        ) -> None:
            self.taxfree_after_period = taxfree_after_period
            self.main_currency = main_currency
            self.use_asset_collections_in_cost_basis = use_asset_collections_in_cost_basis
            self.cost_basis_method = cost_basis_method

    module.DBSettings = DBSettings
    return module


def build_accounting_errors_module() -> types.ModuleType:
    """Build the minimal `rotkehlchen.errors.misc` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.errors.misc")

    class AccountingError(Exception):
        """Stub accounting error type.

        Authored by: OpenCode
        """

    module.AccountingError = AccountingError
    return module


def build_logging_module() -> types.ModuleType:
    """Build the minimal `rotkehlchen.logging` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.logging")

    class RotkehlchenLogsAdapter:
        """Stub log adapter used by the cost-basis module.

        Authored by: OpenCode
        """

        def __init__(self, logger: object) -> None:
            self.logger = logger

        def debug(self, *args: object, **kwargs: object) -> None:
            return None

        def error(self, *args: object, **kwargs: object) -> None:
            return None

    module.RotkehlchenLogsAdapter = RotkehlchenLogsAdapter
    return module


def build_deserialize_module(fval_class: type) -> types.ModuleType:
    """Build the minimal `rotkehlchen.serialization.deserialize` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.serialization.deserialize")

    def deserialize_fval(value: str, name: str, location: str) -> object:
        """Deserialize one decimal value for the cost-basis module.

        Authored by: OpenCode
        """

        _ = name, location
        return fval_class(value)

    module.deserialize_fval = deserialize_fval
    return module


def build_types_module(fval_class: type) -> types.ModuleType:
    """Build the minimal `rotkehlchen.types` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.types")

    class CostBasisMethod(enum.Enum):
        """Stub rotki cost-basis enum.

        Authored by: OpenCode
        """

        FIFO = "FIFO"
        LIFO = "LIFO"
        HIFO = "HIFO"
        ACB = "ACB"

    class Location(enum.Enum):
        """Stub rotki location enum.

        Authored by: OpenCode
        """

        BLOCKCHAIN = "blockchain"

    class Timestamp(int):
        """Stub timestamp type.

        Authored by: OpenCode
        """

    module.CostBasisMethod = CostBasisMethod
    module.Location = Location
    module.Price = fval_class
    module.Timestamp = Timestamp
    return module


def build_user_messages_module() -> types.ModuleType:
    """Build the minimal `rotkehlchen.user_messages` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.user_messages")

    class MessagesAggregator:
        """Stub message aggregator.

        Authored by: OpenCode
        """

    module.MessagesAggregator = MessagesAggregator
    return module


def build_customizable_date_module() -> types.ModuleType:
    """Build the minimal `rotkehlchen.utils.mixins.customizable_date` stub.

    Authored by: OpenCode
    """
    module = types.ModuleType("rotkehlchen.utils.mixins.customizable_date")
    db_settings = build_db_settings_module().DBSettings

    class CustomizableDateMixin:
        """Stub mixin providing the settings field and timestamp formatter.

        Authored by: OpenCode
        """

        def __init__(self, database: object | None = None) -> None:
            _ = database
            self.settings = db_settings()

        def timestamp_to_date(self, timestamp: int) -> str:
            return str(timestamp)

    module.CustomizableDateMixin = CustomizableDateMixin
    return module


def execute_rotki_boundary(rotki: dict[str, object], payload: dict[str, object], method_name: str) -> dict[str, object]:
    """Execute one generated oracle input against the loaded rotki boundary.

    Authored by: OpenCode
    """
    cost_basis_method = resolve_cost_basis_method(rotki["CostBasisMethod"], method_name)
    events = rotki["CostBasisEvents"](cost_basis_method)
    manager = events.acquisitions_manager
    settings = rotki["DBSettings"](cost_basis_method=cost_basis_method)
    fval_class = rotki["FVal"]
    timestamp_class = rotki["Timestamp"]
    asset_class = rotki["Asset"]
    acquisition_event_class = rotki["AssetAcquisitionEvent"]
    asset_identifier = str(payload["asset_identity_key"])
    comparison_source_ids = set(str(value) for value in payload["comparison_activity_source_ids"])
    acquisition_source_ids: dict[int, str] = {}
    realized = fval_class(0)
    allocated = fval_class(0)
    open_quantity = fval_class(0)
    open_basis = fval_class(0)
    last_relevant_closing_quantity: object | None = None
    last_relevant_closing_basis: object | None = None
    matches: list[dict[str, str]] = []

    for index, activity in enumerate(payload["activities"], start=1):
        activity_type = str(activity["activity_type"]).upper()
        source_id = str(activity["source_id"])
        quantity = fval_class(activity["quantity"])
        fee_amount = fval_class(activity.get("fee_amount", "0") or "0")
        gross_value = fval_class(activity.get("gross_value", "0") or "0")
        timestamp = timestamp_class(index)

        if activity_type == "BUY":
            acquisition_source_ids[index] = source_id
            acquisition_basis = gross_value + fee_amount
            rate = acquisition_basis / quantity
            manager.add_in_event(
                acquisition_event_class(
                    amount=quantity,
                    timestamp=timestamp,
                    rate=rate,
                    index=index,
                ),
            )
            open_quantity += quantity
            open_basis += acquisition_basis
            if source_id in comparison_source_ids:
                last_relevant_closing_quantity = open_quantity
                last_relevant_closing_basis = open_basis
            continue

        if activity_type != "SELL":
            raise RuntimeError(f"Unsupported activity_type {activity_type}")

        info = manager.calculate_spend_cost_basis(
            quantity,
            asset_class(asset_identifier),
            timestamp,
            [],
            [],
            settings,
            str,
        )
        net_proceeds = gross_value - fee_amount
        total_basis = info.taxable_bought_cost + info.taxfree_bought_cost
        open_quantity -= quantity
        open_basis -= total_basis

        if source_id in comparison_source_ids:
            realized += net_proceeds - total_basis
            allocated += total_basis

            if method_name == "average_cost":
                matches.append(
                    {
                        "disposed_source_id": source_id,
                        "acquisition_source_id": asset_identifier,
                        "matched_quantity": str(quantity),
                        "matched_basis": str(total_basis),
                        "matched_proceeds": str(net_proceeds),
                        "matched_gain_or_loss": str(net_proceeds - total_basis),
                        "support_label": "rotki_backed",
                    },
                )
            else:
                proceeds_per_unit = net_proceeds / quantity
                for match in info.matched_acquisitions:
                    if match.amount == fval_class(0):
                        continue
                    matched_basis = match.event.rate * match.amount
                    matches.append(
                        {
                            "disposed_source_id": source_id,
                            "acquisition_source_id": acquisition_source_ids.get(match.event.index, ""),
                            "matched_quantity": str(match.amount),
                            "matched_basis": str(matched_basis),
                            "matched_proceeds": str(proceeds_per_unit * match.amount),
                            "matched_gain_or_loss": str((proceeds_per_unit * match.amount) - matched_basis),
                            "support_label": "rotki_backed",
                        },
                    )

            last_relevant_closing_quantity = open_quantity
            last_relevant_closing_basis = open_basis

    if last_relevant_closing_quantity is None or last_relevant_closing_basis is None:
        last_relevant_closing_quantity = open_quantity
        last_relevant_closing_basis = open_basis

    return {
        "values": {
            "realized_gain_or_loss": str(realized),
            "allocated_basis": str(allocated),
            "closing_quantity": str(last_relevant_closing_quantity),
            "closing_basis": str(last_relevant_closing_basis),
        },
        "matches": matches,
    }


def resolve_cost_basis_method(cost_basis_method_enum: object, method_name: str) -> object:
    """Map one adapter method name to the loaded rotki enum value.

    Authored by: OpenCode
    """
    method_map = {
        "fifo": cost_basis_method_enum.FIFO,
        "lifo": cost_basis_method_enum.LIFO,
        "hifo": cost_basis_method_enum.HIFO,
        "average_cost": cost_basis_method_enum.ACB,
    }
    if method_name not in method_map:
        raise RuntimeError(f"Unsupported rotki method {method_name}")
    return method_map[method_name]


def calculate_open_state(fval_class: type, manager: object) -> tuple[object, object]:
    """Calculate the remaining open quantity and basis from the rotki manager.

    Authored by: OpenCode
    """
    open_quantity = fval_class(0)
    open_basis = fval_class(0)
    for entry in manager.get_acquisitions():
        open_quantity += entry.remaining_amount
        open_basis += entry.remaining_amount * entry.rate
    return open_quantity, open_basis


if __name__ == "__main__":
    raise SystemExit(main())
