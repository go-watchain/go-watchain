// Copyright 2015 The go-ethereum Authors
// This file is part of the go-watereum library.
//
// The go-watereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-watereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-watereum library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/watchain/go-watchain/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("wat/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("wat/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("wat/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("wat/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("wat/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("wat/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("wat/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("wat/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("wat/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("wat/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("wat/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("wat/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("wat/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("wat/downloader/states/drop", nil)
)
