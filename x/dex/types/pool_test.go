package types

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGetPoolShareBaseDenom(t *testing.T) {
	require.Equal(t, "nibiru/pool/123", GetPoolShareBaseDenom(123))
}

func TestGetPoolShareDisplayDenom(t *testing.T) {
	require.Equal(t, "NIBIRU-POOL-123", GetPoolShareDisplayDenom(123))
}

func TestGetAddress(t *testing.T) {
	tests := []struct {
		name        string
		pool        Pool
		expectPanic bool
	}{
		{
			name: "empty address",
			pool: Pool{
				Address: "",
			},
			expectPanic: true,
		},
		{
			name: "invalid address",
			pool: Pool{
				Address: "asdf",
			},
			expectPanic: true,
		},
		{
			name: "valid address",
			pool: Pool{
				Address: testutil.AccAddress().String(),
			},
			expectPanic: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectPanic {
				require.Panics(t, func() {
					tc.pool.GetAddress()
				})
			} else {
				require.NotPanics(t, func() {
					tc.pool.GetAddress()
				})
			}
		})
	}
}

func TestNewPool(t *testing.T) {
	poolAccountAddr := testutil.AccAddress()
	poolParams := PoolParams{
		SwapFee: sdk.NewDecWithPrec(3, 2),
		ExitFee: sdk.NewDecWithPrec(3, 2),
	}
	poolAssets := []PoolAsset{
		{
			Token:  sdk.NewInt64Coin("foo", 100),
			Weight: sdk.NewInt(1),
		},
		{
			Token:  sdk.NewInt64Coin("bar", 100),
			Weight: sdk.NewInt(1),
		},
	}

	pool, err := NewPool(1 /*=poold*/, poolAccountAddr, poolParams, poolAssets)
	require.NoError(t, err)
	require.Equal(t, Pool{
		Id:         1,
		Address:    poolAccountAddr.String(),
		PoolParams: poolParams,
		PoolAssets: []PoolAsset{
			{
				Token:  sdk.NewInt64Coin("bar", 100),
				Weight: sdk.NewInt(1 << 30),
			},
			{
				Token:  sdk.NewInt64Coin("foo", 100),
				Weight: sdk.NewInt(1 << 30),
			},
		},
		TotalWeight: sdk.NewInt(2 << 30),
		TotalShares: sdk.NewCoin("nibiru/pool/1", sdk.NewIntWithDecimal(100, 18)),
	}, pool)
}

func TestJoinPoolHappyPath(t *testing.T) {
	for _, tc := range []struct {
		name              string
		pool              Pool
		tokensIn          sdk.Coins
		expectedNumShares sdk.Int
		expectedRemCoins  sdk.Coins
		expectedPool      Pool
	}{
		{
			name: "all coins deposited",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 20),
			),
			expectedNumShares: sdk.NewInt(10),
			expectedRemCoins:  sdk.NewCoins(),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 110),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 220),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 110),
			},
		},
		{
			name: "partial coins deposited",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 10),
			),
			expectedNumShares: sdk.NewInt(5),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 5),
			),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 105),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 210),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 105),
			},
		},
		{
			name: "difficult numbers",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 3_498_579),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 1_403_945),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1_000_000),
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 4859), // 0.138885 % of pool
				sdk.NewInt64Coin("bbb", 1345), // 0.09580147 % of pool
			),
			expectedNumShares: sdk.NewInt(958),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 1507),
			),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 3_501_931),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 1_405_290),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1_000_958),
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			numShares, remCoins, err := tc.pool.AddTokensToPool(tc.tokensIn)
			require.NoError(t, err)
			require.Equal(t, tc.expectedNumShares, numShares)
			require.Equal(t, tc.expectedRemCoins, remCoins)
			require.Equal(t, tc.expectedPool, tc.pool)
		})
	}
}

func TestJoinPoolAllTokens(t *testing.T) {
	for _, tc := range []struct {
		name              string
		pool              Pool
		tokensIn          sdk.Coins
		expectedNumShares sdk.Int
		expectedRemCoins  sdk.Coins
		expectedPool      Pool
	}{
		{
			name: "all coins deposited",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 100),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 200),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{SwapFee: sdk.ZeroDec()},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 20),
			),
			expectedNumShares: sdk.NewInt(10),
			expectedRemCoins:  sdk.NewCoins(),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 110),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 220),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 110),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{SwapFee: sdk.ZeroDec()},
			},
		},
		{
			name: "partial coins deposited",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 100),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 200),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{SwapFee: sdk.ZeroDec()},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
				sdk.NewInt64Coin("bbb", 10),
			),
			expectedNumShares: sdk.NewInt(6),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 1),
			),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 109),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 210),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 106),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{SwapFee: sdk.ZeroDec()},
			},
		},
		{
			name: "difficult numbers",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 3_498_579),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1_403_945),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1_000_000),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{SwapFee: sdk.ZeroDec()},
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 4859), // 0.138885 % of pool
				sdk.NewInt64Coin("bbb", 1345), // 0.09580147 % of pool
			),
			expectedNumShares: sdk.NewInt(1172),
			expectedRemCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 3),
			),
			expectedPool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token:  sdk.NewInt64Coin("aaa", 3_503_435),
						Weight: sdk.NewInt(1 << 30),
					},
					{
						Token:  sdk.NewInt64Coin("bbb", 1_405_290),
						Weight: sdk.NewInt(1 << 30),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 1_001_172),
				TotalWeight: sdk.NewInt(2 << 30),
				PoolParams:  PoolParams{SwapFee: sdk.ZeroDec()},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			numShares, remCoins, err := tc.pool.AddAllTokensToPool(tc.tokensIn)
			require.NoError(t, err)
			require.Equal(t, tc.expectedNumShares, numShares)
			require.Equal(t, tc.expectedRemCoins, remCoins)
			require.Equal(t, tc.expectedPool, tc.pool)
		})
	}
}

func TestJoinPoolInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name     string
		pool     Pool
		tokensIn sdk.Coins
	}{
		{
			name: "not enough tokens",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
			},
			tokensIn: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 10),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := tc.pool.AddTokensToPool(tc.tokensIn)
			require.Error(t, err)
		})
	}
}

func TestExitPoolHappyPath(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		pool                    Pool
		exitingShares           sdk.Coin
		expectedCoins           sdk.Coins
		expectedRemainingShares sdk.Coin
		expectedExitedCoins     sdk.Coins
	}{
		{
			name: "all coins withdrawn, no exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams: PoolParams{
					ExitFee: sdk.ZeroDec(),
				},
			},
			exitingShares:           sdk.NewInt64Coin("nibiru/pool/1", 100),
			expectedRemainingShares: sdk.NewInt64Coin("nibiru/pool/1", 0),
			expectedCoins:           nil,
			expectedExitedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 100),
				sdk.NewInt64Coin("bbb", 200),
			),
		},
		{
			name: "all coins withdrawn, exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams: PoolParams{
					ExitFee: sdk.MustNewDecFromStr("0.5"),
				},
			},
			exitingShares:           sdk.NewInt64Coin("nibiru/pool/1", 100),
			expectedRemainingShares: sdk.NewInt64Coin("nibiru/pool/1", 0),
			expectedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 100),
			),
			expectedExitedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 100),
			),
		},
		{
			name: "some coins withdrawn, no exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams: PoolParams{
					ExitFee: sdk.ZeroDec(),
				},
			},
			exitingShares:           sdk.NewInt64Coin("nibiru/pool/1", 50),
			expectedRemainingShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
			expectedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 100),
			),
			expectedExitedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 50),
				sdk.NewInt64Coin("bbb", 100),
			),
		},
		{
			name: "some coins withdrawn, exit fee",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 100),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 200),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 100),
				PoolParams: PoolParams{
					ExitFee: sdk.MustNewDecFromStr("0.5"),
				},
			},
			exitingShares:           sdk.NewInt64Coin("nibiru/pool/1", 50),
			expectedRemainingShares: sdk.NewInt64Coin("nibiru/pool/1", 50),
			expectedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 75),
				sdk.NewInt64Coin("bbb", 150),
			),
			expectedExitedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 25),
				sdk.NewInt64Coin("bbb", 50),
			),
		},
		{
			name: "real numbers",
			pool: Pool{
				PoolAssets: []PoolAsset{
					{
						Token: sdk.NewInt64Coin("aaa", 34_586_245),
					},
					{
						Token: sdk.NewInt64Coin("bbb", 65_469_884),
					},
				},
				TotalShares: sdk.NewInt64Coin("nibiru/pool/1", 2_347_652),
				PoolParams: PoolParams{
					ExitFee: sdk.MustNewDecFromStr("0.003"),
				},
			},
			exitingShares:           sdk.NewInt64Coin("nibiru/pool/1", 74_747),
			expectedRemainingShares: sdk.NewInt64Coin("nibiru/pool/1", 2_272_905),
			expectedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 33_488_356),
				sdk.NewInt64Coin("bbb", 63_391_639),
			),
			expectedExitedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 1_097_889),
				sdk.NewInt64Coin("bbb", 2_078_245),
			),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			exitedCoins, err := tc.pool.ExitPool(tc.exitingShares.Amount)
			require.NoError(t, err)
			require.Equal(t, tc.expectedCoins, tc.pool.PoolBalances())
			// Comparing zero initialized sdk.Int with zero value sdk.Int leads to different results
			if tc.expectedRemainingShares.IsZero() {
				require.True(t, tc.pool.TotalShares.IsZero())
			} else {
				require.Equal(t, tc.expectedRemainingShares, tc.pool.TotalShares)
			}
			require.Equal(t, tc.expectedExitedCoins, exitedCoins)
		})
	}
}

// helper function to create dummy test pools
func MockPool(assets []PoolAsset) Pool {
	return Pool{
		Id: 1,
		PoolParams: PoolParams{
			SwapFee: sdk.SmallestDec(),
			ExitFee: sdk.SmallestDec(),
		},
		PoolAssets:  assets,
		TotalShares: sdk.NewInt64Coin(GetPoolShareBaseDenom(1), 100),
		TotalWeight: sdk.NewInt(2),
	}
}

func TestUpdatePoolAssetTokens(t *testing.T) {
	for _, tc := range []struct {
		name               string
		poolAssets         []PoolAsset
		newAssets          sdk.Coins
		expectedPoolAssets []PoolAsset
	}{
		{
			name: "update pool asset balances",
			poolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 100),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 200),
				},
			},
			newAssets: sdk.NewCoins(
				sdk.NewInt64Coin("aaa", 150),
				sdk.NewInt64Coin("bbb", 125),
			),
			expectedPoolAssets: []PoolAsset{
				{
					Token: sdk.NewInt64Coin("aaa", 150),
				},
				{
					Token: sdk.NewInt64Coin("bbb", 125),
				},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pool := MockPool(tc.poolAssets)
			require.NoError(t, pool.updatePoolAssetBalances(tc.newAssets))
			require.Equal(t, tc.expectedPoolAssets, pool.PoolAssets)
		})
	}
}
