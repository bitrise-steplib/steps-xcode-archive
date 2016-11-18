package exportoptions

import (
	"fmt"
	"path/filepath"

	plist "github.com/DHowett/go-plist"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
)

// ExportOptions ...
type ExportOptions interface {
	Hash() map[string]interface{}
	String() (string, error)
	WriteToFile(pth string) error
	WriteToTmpFile() (string, error)
}

// AppStoreOptionsModel ...
type AppStoreOptionsModel struct {
	TeamID string

	// for app-store exports
	UploadBitcode bool
	UploadSymbols bool
}

// NewAppStoreOptions ...
func NewAppStoreOptions() AppStoreOptionsModel {
	return AppStoreOptionsModel{
		UploadBitcode: UploadBitcodeDefault,
		UploadSymbols: UploadSymbolsDefault,
	}
}

// Hash ...
func (options AppStoreOptionsModel) Hash() map[string]interface{} {
	hash := map[string]interface{}{}
	hash[MethodKey] = MethodAppStore
	if options.TeamID != "" {
		hash[TeamIDKey] = options.TeamID
	}
	if options.UploadBitcode != UploadBitcodeDefault {
		hash[UploadBitcodeKey] = options.UploadBitcode
	}
	if options.UploadSymbols != UploadSymbolsDefault {
		hash[UploadSymbolsKey] = options.UploadSymbols
	}
	return hash
}

// String ...
func (options AppStoreOptionsModel) String() (string, error) {
	hash := options.Hash()
	plistBytes, err := plist.MarshalIndent(hash, plist.XMLFormat, "\t")
	if err != nil {
		return "", fmt.Errorf("failed to marshal export options model, error: %s", err)
	}
	return string(plistBytes), err
}

// WriteToFile ...
func (options AppStoreOptionsModel) WriteToFile(pth string) error {
	return WritePlistToFile(options.Hash(), pth)
}

// WriteToTmpFile ...
func (options AppStoreOptionsModel) WriteToTmpFile() (string, error) {
	return WritePlistToTmpFile(options.Hash())
}

// NonAppStoreOptionsModel ...
type NonAppStoreOptionsModel struct {
	Method Method
	TeamID string

	// for non app-store exports
	CompileBitcode                           bool
	EmbedOnDemandResourcesAssetPacksInBundle bool
	ICloudContainerEnvironment               ICloudContainerEnvironment
	Manifest                                 Manifest
	OnDemandResourcesAssetPacksBaseURL       string
	Thinning                                 string
}

// NewNonAppStoreOptions ...
func NewNonAppStoreOptions(method Method) NonAppStoreOptionsModel {
	return NonAppStoreOptionsModel{
		Method:                                   method,
		CompileBitcode:                           CompileBitcodeDefault,
		EmbedOnDemandResourcesAssetPacksInBundle: EmbedOnDemandResourcesAssetPacksInBundleDefault,
		ICloudContainerEnvironment:               ICloudContainerEnvironmentDefault,
		Thinning:                                 ThinningDefault,
	}
}

// Hash ...
func (options NonAppStoreOptionsModel) Hash() map[string]interface{} {
	hash := map[string]interface{}{}
	if options.Method != "" {
		hash[MethodKey] = options.Method
	}
	if options.TeamID != "" {
		hash[TeamIDKey] = options.TeamID
	}
	if options.CompileBitcode != CompileBitcodeDefault {
		hash[CompileBitcodeKey] = options.CompileBitcode
	}
	if options.EmbedOnDemandResourcesAssetPacksInBundle != EmbedOnDemandResourcesAssetPacksInBundleDefault {
		hash[EmbedOnDemandResourcesAssetPacksInBundleKey] = options.EmbedOnDemandResourcesAssetPacksInBundle
	}
	if options.ICloudContainerEnvironment != ICloudContainerEnvironmentDefault {
		hash[ICloudContainerEnvironmentKey] = options.ICloudContainerEnvironment
	}
	if !options.Manifest.IsEmpty() {
		hash[ManifestKey] = options.Manifest.ToHash()
	}
	if options.OnDemandResourcesAssetPacksBaseURL != "" {
		hash[OnDemandResourcesAssetPacksBaseURLKey] = options.OnDemandResourcesAssetPacksBaseURL
	}
	if options.Thinning != ThinningDefault {
		hash[ThinningKey] = options.Thinning
	}
	return hash
}

// String ...
func (options NonAppStoreOptionsModel) String() (string, error) {
	hash := options.Hash()
	plistBytes, err := plist.MarshalIndent(hash, plist.XMLFormat, "\t")
	if err != nil {
		return "", fmt.Errorf("failed to marshal export options model, error: %s", err)
	}
	return string(plistBytes), err
}

// WriteToFile ...
func (options NonAppStoreOptionsModel) WriteToFile(pth string) error {
	return WritePlistToFile(options.Hash(), pth)
}

// WriteToTmpFile ...
func (options NonAppStoreOptionsModel) WriteToTmpFile() (string, error) {
	return WritePlistToTmpFile(options.Hash())
}

// WritePlistToFile ...
func WritePlistToFile(options map[string]interface{}, pth string) error {
	plistBytes, err := plist.MarshalIndent(options, plist.XMLFormat, "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal export options model, error: %s", err)
	}
	if err := fileutil.WriteBytesToFile(pth, plistBytes); err != nil {
		return fmt.Errorf("failed to write export options, error: %s", err)
	}

	return nil
}

// WritePlistToTmpFile ...
func WritePlistToTmpFile(options map[string]interface{}) (string, error) {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("output")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir, error: %s", err)
	}
	pth := filepath.Join(tmpDir, "exportOptions.plist")

	if err := WritePlistToFile(options, pth); err != nil {
		return "", fmt.Errorf("failed to write to file options, error: %s", err)
	}

	return pth, nil
}
