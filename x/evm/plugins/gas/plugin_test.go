// SPDX-License-Identifier: BUSL-1.1
//
// Copyright (C) 2023, Berachain Foundation. All rights reserved.
// Use of this software is govered by the Business Source License included
// in the LICENSE file of this repository and at www.mariadb.com/bsl11.
//
// ANY USE OF THE LICENSED WORK IN VIOLATION OF THIS LICENSE WILL AUTOMATICALLY
// TERMINATE YOUR RIGHTS UNDER THIS LICENSE FOR THE CURRENT AND ALL OTHER
// VERSIONS OF THE LICENSED WORK.
//
// THIS LICENSE DOES NOT GRANT YOU ANY RIGHT IN ANY TRADEMARK OR LOGO OF
// LICENSOR OR ITS AFFILIATES (PROVIDED THAT YOU MAY USE A TRADEMARK OR LOGO OF
// LICENSOR AS EXPRESSLY REQUIRED BY THIS LICENSE).
//
// TO THE EXTENT PERMITTED BY APPLICABLE LAW, THE LICENSED WORK IS PROVIDED ON
// AN “AS IS” BASIS. LICENSOR HEREBY DISCLAIMS ALL WARRANTIES AND CONDITIONS,
// EXPRESS OR IMPLIED, INCLUDING (WITHOUT LIMITATION) WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, NON-INFRINGEMENT, AND
// TITLE.

package gas

import (
	"math"

	storetypes "cosmossdk.io/store/types"
	"github.com/berachain/stargazer/lib/utils"
	"github.com/berachain/stargazer/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin", func() {
	var ctx sdk.Context
	var p *plugin
	var blockGasMeter storetypes.GasMeter
	var txGasLimit = uint64(1000)

	BeforeEach(func() {
		// new block
		blockGasMeter = storetypes.NewGasMeter(uint64(2000))
		ctx = testutil.NewContext().WithBlockGasMeter(blockGasMeter)
		p = utils.MustGetAs[*plugin](NewPluginFrom(ctx))
	})

	It("correctly consume, refund, and report cumulative in the same block", func() {
		// tx 1
		err := p.SetGasLimit(txGasLimit)
		Expect(err).To(BeNil())
		err = p.ConsumeGas(500)
		Expect(err).To(BeNil())
		Expect(p.GasUsed()).To(Equal(uint64(500)))
		Expect(p.GasRemaining()).To(Equal(uint64(500)))

		p.RefundGas(250)
		Expect(p.GasUsed()).To(Equal(uint64(250)))
		Expect(p.CumulativeGasUsed()).To(Equal(uint64(250)))
		blockGasMeter.ConsumeGas(250, "") // finalize tx 1

		p.Reset(testutil.NewContext().WithBlockGasMeter(blockGasMeter))

		// tx 2
		err = p.SetGasLimit(txGasLimit)
		Expect(err).To(BeNil())
		Expect(p.CumulativeGasUsed()).To(Equal(uint64(250)))
		err = p.ConsumeGas(1000)
		Expect(err).To(BeNil())
		Expect(p.GasUsed()).To(Equal(uint64(1000)))
		Expect(p.GasRemaining()).To(Equal(uint64(0)))
		Expect(p.CumulativeGasUsed()).To(Equal(uint64(1250)))
		blockGasMeter.ConsumeGas(1000, "") // finalize tx 2

		p.Reset(testutil.NewContext().WithBlockGasMeter(blockGasMeter))

		// tx 3
		err = p.SetGasLimit(txGasLimit)
		Expect(err).To(BeNil())
		Expect(p.CumulativeGasUsed()).To(Equal(uint64(1250)))
		err = p.ConsumeGas(1000) // tx 3 should fail but no error here (250 over block limit)
		Expect(err).To(BeNil())
		Expect(p.GasUsed()).To(Equal(uint64(1000)))
		Expect(p.GasRemaining()).To(Equal(uint64(0)))
		Expect(p.CumulativeGasUsed()).To(Equal(uint64(2000)))             // total is 2250, but capped at 2000
		Expect(func() { blockGasMeter.ConsumeGas(1000, "") }).To(Panic()) // finalize tx 3
	})

	It("should error on overconsumption in tx", func() {
		err := p.SetGasLimit(txGasLimit)
		Expect(err).To(BeNil())
		err = p.ConsumeGas(1000)
		Expect(err).To(BeNil())
		err = p.ConsumeGas(1)
		Expect(err.Error()).To(Equal("out of gas"))
	})

	It("should error on uint64 overflow", func() {
		err := p.SetGasLimit(math.MaxUint64)
		Expect(err).To(BeNil())
		err = p.ConsumeGas(math.MaxUint64)
		Expect(err).To(BeNil())
		err = p.ConsumeGas(1)
		Expect(err.Error()).To(Equal("gas uint64 overflow"))
	})
})