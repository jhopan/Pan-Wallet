"""Pure wallet hold/capture rules; persistence comes later via PostgreSQL."""
from __future__ import annotations

from dataclasses import dataclass
from uuid import uuid4


class WalletRuleError(ValueError):
    pass


@dataclass
class Hold:
    id: str
    product: str
    reference_id: str
    amount: int
    status: str = "active"


class WalletState:
    def __init__(self, balance: int):
        if balance < 0:
            raise WalletRuleError("balance tidak boleh negatif")
        self.balance = balance
        self._holds: list[Hold] = []

    @property
    def available_balance(self) -> int:
        held = sum(hold.amount for hold in self._holds if hold.status == "active")
        return self.balance - held

    def create_hold(self, product: str, reference_id: str, amount: int) -> Hold:
        if not product or not reference_id or amount <= 0:
            raise WalletRuleError("hold tidak valid")
        if amount > self.available_balance:
            raise WalletRuleError("saldo tidak cukup")
        hold = Hold(id=str(uuid4()), product=product, reference_id=reference_id, amount=amount)
        self._holds.append(hold)
        return hold

    def capture_hold(self, hold: Hold) -> None:
        if hold.status != "active":
            raise WalletRuleError("hold sudah diproses")
        hold.status = "captured"
        self.balance -= hold.amount

    def release_hold(self, hold: Hold) -> None:
        if hold.status != "active":
            raise WalletRuleError("hold sudah diproses")
        hold.status = "released"
