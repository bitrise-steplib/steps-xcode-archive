package common

import (
	"slices"

	"github.com/bitrise-io/bitrise-build-cache-cli/v2/internal/consts"
)

const datacenterEnvKey = "BITRISE_DEN_VM_DATACENTER"

//nolint:gochecknoglobals
var (
	// nonRBEDCs are datacenters where RBE is not available
	nonRBEDCs = []string{
		consts.USEAST1,
	}
)

// SelectCacheEndpointURL - if endpointURL provided use that,
// otherwise select the best build cache endpoint automatically
func SelectCacheEndpointURL(endpointURL string, envs map[string]string) string {
	if endpointURL == "" {
		endpointURL = envs["BITRISE_BUILD_CACHE_ENDPOINT"]
	}
	if len(endpointURL) > 0 {
		return endpointURL
	}

	return consts.BitriseAccelerate
}

// SelectRBEEndpointURL - if endpointURL provided use that,
// otherwise select the RBE endpoint from environment
func SelectRBEEndpointURL(endpointURL string, envs map[string]string) string {
	if endpointURL == "" {
		endpointURL = envs["BITRISE_RBE_ENDPOINT"]
	}
	if len(endpointURL) > 0 {
		return endpointURL
	}

	bitriseDenVMDatacenter := envs[datacenterEnvKey]
	if slices.Contains(nonRBEDCs, bitriseDenVMDatacenter) {
		return ""
	}

	return consts.BitriseAccelerate
}
