//go:build !windows

package platform

import (
	"MSMP/agent/common"
	"MSMP/agent/posix"
)

func init() {
	common.CollectAssetInfoFull = posix.CollectAssetInfoFull
	common.CollectMetrics = posix.CollectMetrics
}
