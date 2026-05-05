package consts

const (
	IAD1    = "IAD1"
	ORD1    = "ORD1"
	USEAST1 = "US_EAST1"

	// BitriseAccelerate currently pointing to IAD1, but in the same time, it's environment-aware.
	// It points to the appropriate instance for the respective datacenter when used on VMs managed by Bitrise.
	// More info: https://github.com/bitrise-io/build-prebooting-deployments/blob/production/preboot-reconciler/startup_script_extension_macos_bitvirt.sh#L72
	BitriseAccelerate = "grpcs://bitrise-accelerate.services.bitrise.io"

	XcodeAnalyticsServiceEndpoint         = "https://xcode-analytics.services.bitrise.io"
	MultiplatformAnalyticsServiceEndpoint = "https://multiplatform-analytics.services.bitrise.io"

	BitriseWebsiteBaseURL = "https://app.bitrise.io"

	// Gradle Remote Build Cache related consts
	GradleRemoteBuildCachePluginDepVersion = "1.3.3"

	// Gradle Analytics related consts
	GradleAnalyticsPluginDepVersion = "2.7.1"
	GradleAnalyticsEndpoint         = "gradle-analytics.services.bitrise.io"
	GradleAnalyticsPort             = 443
	GradleAnalyticsHTTPEndpoint     = "https://gradle-sink.services.bitrise.io"
	GradleAnalyticsGRPCEndpoint     = "grpcs://gradle-analytics.services.bitrise.io:444"

	// Gradle Common Plugin version
	GradleCommonPluginDepVersion = "1.0.7"

	// Gradle Test Distribution Plugin version
	GradleTestDistributionPluginDepVersion = "2.2.10"
	GradleTestDistributionEndpoint         = "grpcs://bitrise-accelerate.services.bitrise.io"
	GradleTestDistributionKvEndpoint       = "grpcs://bitrise-accelerate.services.bitrise.io"
	GradleTestDistributionPort             = 443
)
