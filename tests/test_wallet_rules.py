import unittest
from wallet.rules import Hold, WalletState, WalletRuleError


class WalletRulesTest(unittest.TestCase):
    def test_hold_then_capture_reduces_balance_once(self):
        wallet = WalletState(balance=50_000)
        hold = wallet.create_hold("tembak-paket", "TP-1001", 25_000)
        self.assertEqual(wallet.available_balance, 25_000)

        wallet.capture_hold(hold)
        self.assertEqual(wallet.balance, 25_000)
        self.assertEqual(wallet.available_balance, 25_000)

        with self.assertRaises(WalletRuleError):
            wallet.capture_hold(hold)

    def test_hold_rejects_amount_above_available_balance(self):
        wallet = WalletState(balance=20_000)
        wallet.create_hold("agenpulsa", "AP-1001", 15_000)

        with self.assertRaises(WalletRuleError):
            wallet.create_hold("tembak-paket", "TP-1001", 10_000)


if __name__ == "__main__":
    unittest.main()
