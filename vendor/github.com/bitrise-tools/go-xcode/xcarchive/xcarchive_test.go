package xcarchive

import (
	"testing"

	"os"
	"path/filepath"

	"strings"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/stretchr/testify/require"
)

func TestFindEmbeddedMobileProvision(t *testing.T) {
	t.Log("valid dsyms dir path")
	{
		// create test embedded.mobileprovision
		// xyz.xcarchive/Products/Applications/xyz.app/embedded.mobileprovision
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__xcarchive__")
		require.NoError(t, err)

		tmpDirWithSpace := filepath.Join(tmpDir, "sapce test")
		require.NoError(t, os.MkdirAll(tmpDirWithSpace, 0777))

		xcarchiveDirPth := filepath.Join(tmpDirWithSpace, `char [Test]\*.xcarchive`)
		appDirPth := filepath.Join(xcarchiveDirPth, `Products/Applications/char [Test]\*.app`)
		require.NoError(t, os.MkdirAll(appDirPth, 0777))

		embeddedMobileprovisionPth := filepath.Join(appDirPth, "embedded.mobileprovision")
		require.NoError(t, fileutil.WriteStringToFile(embeddedMobileprovisionPth, ""))
		// ---

		pth, err := FindEmbeddedMobileProvision(xcarchiveDirPth)
		require.NoError(t, err)
		require.Equal(t, embeddedMobileprovisionPth, pth)
	}

	t.Log("invalid .app dir path - extra path component")
	{
		// create test embedded.mobileprovision
		// xyz.xcarchive/Products/Applications/xyz.app/invalidcomponent/embedded.mobileprovision
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__xcarchive__")
		require.NoError(t, err)

		tmpDirWithSpace := filepath.Join(tmpDir, "sapce test")
		require.NoError(t, os.MkdirAll(tmpDirWithSpace, 0777))

		xcarchiveDirPth := filepath.Join(tmpDirWithSpace, `char [Test]\*.xcarchive`)
		appDirPth := filepath.Join(xcarchiveDirPth, `Products/Applications/char [Test]\*.app/invalidcomponent`)
		require.NoError(t, os.MkdirAll(appDirPth, 0777))

		embeddedMobileprovisionPth := filepath.Join(appDirPth, "embedded.mobileprovision")
		require.NoError(t, fileutil.WriteStringToFile(embeddedMobileprovisionPth, ""))
		// ---

		pth, err := FindEmbeddedMobileProvision(xcarchiveDirPth)
		require.EqualError(t, err, "no embedded.mobileprovision found")
		require.Equal(t, "", pth)
	}

	t.Log("invalid .app dir path - missing path component")
	{
		// create test embedded.mobileprovision
		// xyz.xcarchive/Products/Applications/embedded.mobileprovision
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__xcarchive__")
		require.NoError(t, err)

		tmpDirWithSpace := filepath.Join(tmpDir, "sapce test")
		require.NoError(t, os.MkdirAll(tmpDirWithSpace, 0777))

		xcarchiveDirPth := filepath.Join(tmpDirWithSpace, `char [Test]\*.xcarchive`)
		appDirPth := filepath.Join(xcarchiveDirPth, `Products/Applications`)
		require.NoError(t, os.MkdirAll(appDirPth, 0777))

		embeddedMobileprovisionPth := filepath.Join(appDirPth, "embedded.mobileprovision")
		require.NoError(t, fileutil.WriteStringToFile(embeddedMobileprovisionPth, ""))
		// ---

		pth, err := FindEmbeddedMobileProvision(xcarchiveDirPth)
		require.EqualError(t, err, "no embedded.mobileprovision found")
		require.Equal(t, "", pth)
	}
}

func TestFindDSYMs(t *testing.T) {
	t.Log("valid dsyms dir path")
	{
		// create test dsyms
		// xyz.xcarchive/dSYMs/xyz.app.dSYM
		// xyz.xcarchive/dSYMs/framework.dSYM
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__xcarchive__")
		require.NoError(t, err)

		tmpDirWithSpace := filepath.Join(tmpDir, "sapce test")
		require.NoError(t, os.MkdirAll(tmpDirWithSpace, 0777))

		xcarchiveDirPth := filepath.Join(tmpDirWithSpace, `char [Test]\*.xcarchive`)
		dsymsDirPth := filepath.Join(xcarchiveDirPth, `dSYMs`)
		require.NoError(t, os.MkdirAll(dsymsDirPth, 0777))

		appDsymPth := filepath.Join(dsymsDirPth, `char [Test]\*.app.dSYM`)
		require.NoError(t, fileutil.WriteStringToFile(appDsymPth, ""))

		frameworkDsymPth := filepath.Join(dsymsDirPth, `framework.dSYM`)
		require.NoError(t, fileutil.WriteStringToFile(frameworkDsymPth, ""))
		// ---

		appDsym, frameworkDsyms, err := FindDSYMs(xcarchiveDirPth)
		require.NoError(t, err)
		require.Equal(t, appDsymPth, appDsym)
		require.Equal(t, 1, len(frameworkDsyms))
		require.Equal(t, frameworkDsymPth, frameworkDsyms[0])
	}

	t.Log("invalid .app dir path - extra path component")
	{
		// create test dsyms
		// xyz.xcarchive/dSYMs/invalidcomponent/xyz.app.dSYM
		// xyz.xcarchive/dSYMs/invalidcomponent/framework.dSYM
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__xcarchive__")
		require.NoError(t, err)

		tmpDirWithSpace := filepath.Join(tmpDir, "sapce test")
		require.NoError(t, os.MkdirAll(tmpDirWithSpace, 0777))

		xcarchiveDirPth := filepath.Join(tmpDirWithSpace, `char [Test]\*.xcarchive`)
		dsymsDirPth := filepath.Join(xcarchiveDirPth, `dSYMs/invalidcomponent`)
		require.NoError(t, os.MkdirAll(dsymsDirPth, 0777))

		appDsymPth := filepath.Join(dsymsDirPth, `char [Test]\*.app.dSYM`)
		require.NoError(t, fileutil.WriteStringToFile(appDsymPth, ""))

		frameworkDsymPth := filepath.Join(dsymsDirPth, `framework.dSYM`)
		require.NoError(t, fileutil.WriteStringToFile(frameworkDsymPth, ""))
		// ---

		appDsym, frameworkDsyms, err := FindDSYMs(xcarchiveDirPth)
		require.EqualError(t, err, "no dsym found")
		require.Equal(t, "", appDsym)
		require.Equal(t, 0, len(frameworkDsyms))
	}

	t.Log("invalid .app dir path - missing path component")
	{
		// create test dsyms
		// xyz.xcarchive/xyz.app.dSYM
		// xyz.xcarchive/framework.dSYM
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__xcarchive__")
		require.NoError(t, err)

		tmpDirWithSpace := filepath.Join(tmpDir, "sapce test")
		require.NoError(t, os.MkdirAll(tmpDirWithSpace, 0777))

		xcarchiveDirPth := filepath.Join(tmpDirWithSpace, `char [Test]\*.xcarchive`)
		dsymsDirPth := xcarchiveDirPth
		require.NoError(t, os.MkdirAll(dsymsDirPth, 0777))

		appDsymPth := filepath.Join(dsymsDirPth, `char [Test]\*.app.dSYM`)
		require.NoError(t, fileutil.WriteStringToFile(appDsymPth, ""))

		frameworkDsymPth := filepath.Join(dsymsDirPth, `framework.dSYM`)
		require.NoError(t, fileutil.WriteStringToFile(frameworkDsymPth, ""))
		// ---

		appDsym, frameworkDsyms, err := FindDSYMs(xcarchiveDirPth)
		require.Equal(t, true, strings.Contains(err.Error(), "no such file or directory"))
		require.Equal(t, "", appDsym)
		require.Equal(t, 0, len(frameworkDsyms))
	}
}

func TestFindApp(t *testing.T) {
	t.Log("valid .app dir path")
	{
		// create test app
		// xyz.xcarchive/Products/Applications/xyz.app
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__xcarchive__")
		require.NoError(t, err)

		tmpDirWithSpace := filepath.Join(tmpDir, "sapce test")
		require.NoError(t, os.MkdirAll(tmpDirWithSpace, 0777))

		xcarchiveDirPth := filepath.Join(tmpDirWithSpace, `char [Test]\*.xcarchive`)
		appDirPth := filepath.Join(xcarchiveDirPth, `Products/Applications/char [Test]\*.app`)
		require.NoError(t, os.MkdirAll(appDirPth, 0777))
		// ---

		pth, err := FindApp(xcarchiveDirPth)
		require.NoError(t, err)
		require.Equal(t, appDirPth, pth)
	}

	t.Log("invalid .app dir path - extra path component")
	{
		// create test app
		// xyz.xcarchive/Products/Applications/invalidcomponent/xyz.app
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__xcarchive__")
		require.NoError(t, err)

		tmpDirWithSpace := filepath.Join(tmpDir, "sapce test")
		require.NoError(t, os.MkdirAll(tmpDirWithSpace, 0777))

		xcarchiveDirPth := filepath.Join(tmpDirWithSpace, `char [Test]\*.xcarchive`)
		appDirPth := filepath.Join(xcarchiveDirPth, `Products/Applications/invalidcomponent/char [Test]\*.app`)
		require.NoError(t, os.MkdirAll(appDirPth, 0777))
		// ---

		pth, err := FindApp(xcarchiveDirPth)
		require.EqualError(t, err, "no app found")
		require.Equal(t, "", pth)
	}

	t.Log("invalid .app dir path - missing path component")
	{
		// create test app
		// xyz.xcarchive/Products/xyz.app
		tmpDir, err := pathutil.NormalizedOSTempDirPath("__xcarchive__")
		require.NoError(t, err)

		tmpDirWithSpace := filepath.Join(tmpDir, "sapce test")
		require.NoError(t, os.MkdirAll(tmpDirWithSpace, 0777))

		xcarchiveDirPth := filepath.Join(tmpDirWithSpace, `char [Test]\*.xcarchive`)
		appDirPth := filepath.Join(xcarchiveDirPth, `Products/char [Test]\*.app`)
		require.NoError(t, os.MkdirAll(appDirPth, 0777))
		// ---

		pth, err := FindApp(xcarchiveDirPth)
		require.Equal(t, true, strings.Contains(err.Error(), "no such file or directory"))
		require.Equal(t, "", pth)
	}
}
