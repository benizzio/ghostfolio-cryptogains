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
from dataclasses import dataclass, field
from pathlib import Path


@dataclass(frozen=True)
class AdapterActivity:
    """Parsed activity row from one generated adapter input.

    Authored by: OpenCode
    """

    source_id: str
    activity_type: str
    quantity: str
    fee_amount: str
    gross_value: str


@dataclass(frozen=True)
class AdapterInput:
    """Parsed adapter input consumed by the rotki boundary.

    Authored by: OpenCode
    """

    asset_identity_key: str
    comparison_activity_source_ids: set[str]
    activities: list[AdapterActivity]


@dataclass
class AdapterMatchEvidence:
    """Normalized match evidence returned to the Go oracle generator.

    Authored by: OpenCode
    """

    disposed_source_id: str
    acquisition_source_id: str
    matched_quantity: str
    matched_basis: str
    matched_proceeds: str
    matched_gain_or_loss: str
    support_label: str = "rotki_backed"

    def to_json(self) -> dict[str, str]:
        """Return the persisted adapter JSON shape for this match.

        Authored by: OpenCode
        """

        return {
            "disposed_source_id": self.disposed_source_id,
            "acquisition_source_id": self.acquisition_source_id,
            "matched_quantity": self.matched_quantity,
            "matched_basis": self.matched_basis,
            "matched_proceeds": self.matched_proceeds,
            "matched_gain_or_loss": self.matched_gain_or_loss,
            "support_label": self.support_label,
        }


@dataclass
class AdapterOutputValues:
    """Aggregate oracle values returned to the Go oracle generator.

    Authored by: OpenCode
    """

    realized_gain_or_loss: str
    allocated_basis: str
    closing_quantity: str
    closing_basis: str

    def to_json(self) -> dict[str, str]:
        """Return the persisted adapter JSON shape for aggregate values.

        Authored by: OpenCode
        """

        return {
            "realized_gain_or_loss": self.realized_gain_or_loss,
            "allocated_basis": self.allocated_basis,
            "closing_quantity": self.closing_quantity,
            "closing_basis": self.closing_basis,
        }


@dataclass
class AdapterOutput:
    """Complete adapter response returned on stdout.

    Authored by: OpenCode
    """

    values: AdapterOutputValues
    matches: list[AdapterMatchEvidence]

    def to_json(self) -> dict[str, object]:
        """Return the persisted adapter JSON response shape.

        Authored by: OpenCode
        """

        return {
            "values": self.values.to_json(),
            "matches": [match.to_json() for match in self.matches],
        }


@dataclass
class OracleExecutionState:
    """Mutable state accumulated while rotki processes adapter activities.

    Authored by: OpenCode
    """

    fval_class: type
    realized: object
    allocated: object
    open_quantity: object
    open_basis: object
    last_relevant_closing_quantity: object | None = None
    last_relevant_closing_basis: object | None = None
    acquisition_source_ids: dict[int, str] = field(default_factory=dict)
    matches: list[AdapterMatchEvidence] = field(default_factory=list)


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

    rotki = load_rotki_boundary(source_root)
    adapter_input = load_adapter_input(input_path)
    output = execute_rotki_boundary(rotki, adapter_input, args.rotki_method)
    json.dump(output.to_json(), sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 0


def load_adapter_input(input_path: Path) -> AdapterInput:
    """Load and parse one generated adapter input file.

    Authored by: OpenCode
    """

    with input_path.open(encoding="utf-8") as handle:
        payload = json.load(handle)

    return parse_adapter_input(payload)


def parse_adapter_input(payload: dict[str, object]) -> AdapterInput:
    """Parse the explicit adapter input fields used by this boundary.

    Authored by: OpenCode
    """

    raw_activities = payload.get("activities")
    if not isinstance(raw_activities, list):
        raise RuntimeError("Adapter input activities must be a list")

    raw_comparison_source_ids = payload.get("comparison_activity_source_ids")
    if not isinstance(raw_comparison_source_ids, list):
        raise RuntimeError("Adapter input comparison_activity_source_ids must be a list")

    return AdapterInput(
        asset_identity_key=str(payload["asset_identity_key"]),
        comparison_activity_source_ids={str(value) for value in raw_comparison_source_ids},
        activities=[parse_adapter_activity(activity) for activity in raw_activities],
    )


def parse_adapter_activity(raw_activity: object) -> AdapterActivity:
    """Parse one explicit adapter activity row.

    Authored by: OpenCode
    """

    if not isinstance(raw_activity, dict):
        raise RuntimeError("Adapter activity must be an object")

    return AdapterActivity(
        source_id=str(raw_activity["source_id"]),
        activity_type=str(raw_activity["activity_type"]).upper(),
        quantity=str(raw_activity["quantity"]),
        fee_amount=str(raw_activity.get("fee_amount", "0") or "0"),
        gross_value=str(raw_activity.get("gross_value", "0") or "0"),
    )


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


def execute_rotki_boundary(rotki: dict[str, object], adapter_input: AdapterInput, method_name: str) -> AdapterOutput:
    """Execute one generated oracle input against the loaded rotki boundary.

    Authored by: OpenCode
    """

    cost_basis_method = resolve_cost_basis_method(rotki["CostBasisMethod"], method_name)
    events = rotki["CostBasisEvents"](cost_basis_method)
    manager = events.acquisitions_manager
    settings = rotki["DBSettings"](cost_basis_method=cost_basis_method)
    state = new_oracle_execution_state(rotki["FVal"])

    for index, activity in enumerate(adapter_input.activities, start=1):
        apply_activity(
            rotki=rotki,
            adapter_input=adapter_input,
            method_name=method_name,
            manager=manager,
            settings=settings,
            state=state,
            index=index,
            activity=activity,
        )

    return build_adapter_response(state)


def new_oracle_execution_state(fval_class: type) -> OracleExecutionState:
    """Create zero-valued oracle execution state.

    Authored by: OpenCode
    """

    return OracleExecutionState(
        fval_class=fval_class,
        realized=fval_class(0),
        allocated=fval_class(0),
        open_quantity=fval_class(0),
        open_basis=fval_class(0),
    )


def apply_activity(
    rotki: dict[str, object],
    adapter_input: AdapterInput,
    method_name: str,
    manager: object,
    settings: object,
    state: OracleExecutionState,
    index: int,
    activity: AdapterActivity,
) -> None:
    """Route one parsed adapter activity to the buy or sell handler.

    Authored by: OpenCode
    """

    quantity = state.fval_class(activity.quantity)
    fee_amount = state.fval_class(activity.fee_amount)
    gross_value = state.fval_class(activity.gross_value)
    timestamp = rotki["Timestamp"](index)

    if activity.activity_type == "BUY":
        record_buy(
            rotki=rotki,
            manager=manager,
            state=state,
            index=index,
            activity=activity,
            quantity=quantity,
            fee_amount=fee_amount,
            gross_value=gross_value,
            timestamp=timestamp,
            comparison_source_ids=adapter_input.comparison_activity_source_ids,
        )
        return

    if activity.activity_type != "SELL":
        raise RuntimeError(f"Unsupported activity_type {activity.activity_type}")

    record_sell(
        rotki=rotki,
        adapter_input=adapter_input,
        method_name=method_name,
        manager=manager,
        settings=settings,
        state=state,
        activity=activity,
        quantity=quantity,
        fee_amount=fee_amount,
        gross_value=gross_value,
        timestamp=timestamp,
    )


def record_buy(
    rotki: dict[str, object],
    manager: object,
    state: OracleExecutionState,
    index: int,
    activity: AdapterActivity,
    quantity: object,
    fee_amount: object,
    gross_value: object,
    timestamp: object,
    comparison_source_ids: set[str],
) -> None:
    """Apply one acquisition to the rotki manager and closing state.

    Authored by: OpenCode
    """

    state.acquisition_source_ids[index] = activity.source_id
    acquisition_basis = gross_value + fee_amount
    rate = acquisition_basis / quantity
    manager.add_in_event(
        rotki["AssetAcquisitionEvent"](
            amount=quantity,
            timestamp=timestamp,
            rate=rate,
            index=index,
        ),
    )
    state.open_quantity += quantity
    state.open_basis += acquisition_basis
    if activity.source_id in comparison_source_ids:
        update_closing_state(state)


def record_sell(
    rotki: dict[str, object],
    adapter_input: AdapterInput,
    method_name: str,
    manager: object,
    settings: object,
    state: OracleExecutionState,
    activity: AdapterActivity,
    quantity: object,
    fee_amount: object,
    gross_value: object,
    timestamp: object,
) -> None:
    """Apply one disposal through rotki and normalize comparable evidence.

    Authored by: OpenCode
    """

    info = manager.calculate_spend_cost_basis(
        quantity,
        rotki["Asset"](adapter_input.asset_identity_key),
        timestamp,
        [],
        [],
        settings,
        str,
    )
    net_proceeds = gross_value - fee_amount
    total_basis = info.taxable_bought_cost + info.taxfree_bought_cost
    state.open_quantity -= quantity
    state.open_basis -= total_basis

    if activity.source_id not in adapter_input.comparison_activity_source_ids:
        return

    state.realized += net_proceeds - total_basis
    state.allocated += total_basis
    append_match_evidence(
        state=state,
        method_name=method_name,
        asset_identifier=adapter_input.asset_identity_key,
        source_id=activity.source_id,
        quantity=quantity,
        net_proceeds=net_proceeds,
        total_basis=total_basis,
        matched_acquisitions=info.matched_acquisitions,
    )
    update_closing_state(state)


def append_match_evidence(
    state: OracleExecutionState,
    method_name: str,
    asset_identifier: str,
    source_id: str,
    quantity: object,
    net_proceeds: object,
    total_basis: object,
    matched_acquisitions: object,
) -> None:
    """Append normalized match evidence for one comparable disposal.

    Authored by: OpenCode
    """

    if method_name == "average_cost":
        state.matches.append(
            AdapterMatchEvidence(
                disposed_source_id=source_id,
                acquisition_source_id=asset_identifier,
                matched_quantity=str(quantity),
                matched_basis=str(total_basis),
                matched_proceeds=str(net_proceeds),
                matched_gain_or_loss=str(net_proceeds - total_basis),
            ),
        )
        return

    proceeds_per_unit = net_proceeds / quantity
    for match in matched_acquisitions:
        if match.amount == state.fval_class(0):
            continue
        matched_basis = match.event.rate * match.amount
        state.matches.append(
            AdapterMatchEvidence(
                disposed_source_id=source_id,
                acquisition_source_id=state.acquisition_source_ids.get(match.event.index, ""),
                matched_quantity=str(match.amount),
                matched_basis=str(matched_basis),
                matched_proceeds=str(proceeds_per_unit * match.amount),
                matched_gain_or_loss=str((proceeds_per_unit * match.amount) - matched_basis),
            ),
        )


def update_closing_state(state: OracleExecutionState) -> None:
    """Capture the latest relevant closing quantity and basis.

    Authored by: OpenCode
    """

    state.last_relevant_closing_quantity = state.open_quantity
    state.last_relevant_closing_basis = state.open_basis


def build_adapter_response(state: OracleExecutionState) -> AdapterOutput:
    """Construct the explicit adapter response object.

    Authored by: OpenCode
    """

    closing_quantity = state.last_relevant_closing_quantity
    closing_basis = state.last_relevant_closing_basis
    if closing_quantity is None or closing_basis is None:
        closing_quantity = state.open_quantity
        closing_basis = state.open_basis

    return AdapterOutput(
        values=AdapterOutputValues(
            realized_gain_or_loss=str(state.realized),
            allocated_basis=str(state.allocated),
            closing_quantity=str(closing_quantity),
            closing_basis=str(closing_basis),
        ),
        matches=state.matches,
    )


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


if __name__ == "__main__":
    raise SystemExit(main())
