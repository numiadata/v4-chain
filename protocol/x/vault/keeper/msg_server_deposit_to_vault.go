package keeper

import (
	"context"

	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/lib/log"
	"github.com/dydxprotocol/v4-chain/protocol/lib/metrics"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	sendingtypes "github.com/dydxprotocol/v4-chain/protocol/x/sending/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/vault/types"
)

// DepositToVault deposits from a subaccount to a vault.
func (k msgServer) DepositToVault(
	goCtx context.Context,
	msg *types.MsgDepositToVault,
) (*types.MsgDepositToVaultResponse, error) {
	ctx := lib.UnwrapSDKContext(goCtx, types.ModuleName)

	// Mint shares for the vault.
	err := k.MintShares(
		ctx,
		*msg.VaultId,
		msg.SubaccountId.Owner,
		msg.QuoteQuantums.BigInt(),
	)
	if err != nil {
		return nil, err
	}

	// Transfer from sender subaccount to vault.
	// Note: Transfer should take place after minting shares for
	// shares calculation to be correct.
	err = k.sendingKeeper.ProcessTransfer(
		ctx,
		&sendingtypes.Transfer{
			Sender:    *msg.SubaccountId,
			Recipient: *msg.VaultId.ToSubaccountId(),
			AssetId:   assettypes.AssetUsdc.Id,
			Amount:    msg.QuoteQuantums.BigInt().Uint64(),
		},
	)
	if err != nil {
		return nil, err
	}

	// Emit metric on vault equity.
	equity, err := k.GetVaultEquity(ctx, *msg.VaultId)
	if err != nil {
		log.ErrorLogWithError(ctx, "Failed to get vault equity", err, "vaultId", *msg.VaultId)
	} else {
		msg.VaultId.SetGaugeWithLabels(
			metrics.VaultEquity,
			float32(equity.Int64()),
		)
	}

	return &types.MsgDepositToVaultResponse{}, nil
}