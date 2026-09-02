//go:build windows

package platform

import (
	"MSMP/agent/common"
	"MSMP/agent/win"
)

func init() {
	common.CollectAssetInfoFull = win.CollectAssetInfoFull
	common.CollectMetrics = win.CollectMetrics
}
