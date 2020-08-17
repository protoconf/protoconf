package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	assert "github.com/stretchr/testify/require"
)

var c *Compiler

func setup() {
	tmpDir, err := ioutil.TempDir("", "compiler_test")
	if err != nil {
		panic(err)
	}
	err = CopyDirectory("testdata", tmpDir)
	if err != nil {
		panic(err)
	}
	c = NewCompiler(tmpDir, true)

}
func Test(t *testing.T)          { assert.NoError(t, c.CompileFile("test.pconf")) }
func TestValidator(t *testing.T) { assert.Error(t, c.CompileFile("validator_test.pconf")) }
func TestValidatorRepeated(t *testing.T) {
	assert.Error(t, c.CompileFile("validator_repeated_test.pconf"))
}
func TestValidatorMap(t *testing.T) { assert.Error(t, c.CompileFile("validator_map_test.pconf")) }
func TestValidatorPassing(t *testing.T) {
	assert.NoError(t, c.CompileFile("validator_passing_test.pconf"))
}
func TestEnum(t *testing.T)          { assert.NoError(t, c.CompileFile("enum_test.pconf")) }
func TestEnumWrongType(t *testing.T) { assert.Error(t, c.CompileFile("enum_wrong_types_test.pconf")) }
func TestMultiOutputs(t *testing.T)  { assert.NoError(t, c.CompileFile("multioutputs_test.mpconf")) }
func TestIncludePinc(t *testing.T)   { assert.NoError(t, c.CompileFile("include_pinc_test.pconf")) }
func TestLoadMutable(t *testing.T)   { assert.NoError(t, c.CompileFile("load_mutable_test.pconf")) }
func TestAny(t *testing.T)           { assert.NoError(t, c.CompileFile("field_type_any_test.pconf")) }
func TestAnyMissingPackage(t *testing.T) {
	assert.NoError(t, c.CompileFile("field_type_any_from_missing_package_test.pconf"))
}
func TestUinitializedMessage(t *testing.T) {
	assert.NoError(t, c.CompileFile("uninitialized_msg_test.pconf"))
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func CopyDirectory(scrDir, dest string) error {
	entries, err := ioutil.ReadDir(scrDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(scrDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("failed to get raw syscall.Stat_t data for '%s'", sourcePath)
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			if err := CreateIfNotExists(destPath, 0755); err != nil {
				return err
			}
			if err := CopyDirectory(sourcePath, destPath); err != nil {
				return err
			}
		case os.ModeSymlink:
			if err := CopySymLink(sourcePath, destPath); err != nil {
				return err
			}
		default:
			if err := Copy(sourcePath, destPath); err != nil {
				return err
			}
		}

		if err := os.Lchown(destPath, int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}

		isSymlink := entry.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err := os.Chmod(destPath, entry.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func Copy(srcFile, dstFile string) error {
	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}

	defer out.Close()

	in, err := os.Open(srcFile)
	defer in.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return nil
}

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

func CreateIfNotExists(dir string, perm os.FileMode) error {
	if Exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}

func CopySymLink(source, dest string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
}
