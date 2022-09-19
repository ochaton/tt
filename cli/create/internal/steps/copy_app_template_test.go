package steps

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tarantool/tt/cli/cmdcontext"
)

const subdirName = "subdir"

func createArchive(buf io.Writer, files ...string) error {
	gzipWriter := gzip.NewWriter(buf)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, fileName := range files {
		err := addToArchive(tarWriter, fileName)
		if err != nil {
			return fmt.Errorf("Error adding %s to archive: %s", fileName, err)
		}
	}

	return nil
}

func addToArchive(tarWriter *tar.Writer, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	tarHeader, err := tar.FileInfoHeader(stat, stat.Name())
	if err != nil {
		return err
	}

	err = tarWriter.WriteHeader(tarHeader)
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	if err != nil {
		return err
	}

	return nil
}

func TestCopyTemplateDirectory(t *testing.T) {
	dstDir, err := ioutil.TempDir("", testWorkDirName)
	require.NoError(t, err)
	defer os.RemoveAll(dstDir)

	workDir2, err := ioutil.TempDir("", testWorkDirName)
	require.NoError(t, err)
	defer os.RemoveAll(workDir2)

	require.Nil(t, copy.Copy("testdata/copy_template", workDir2))

	var createCtx cmdcontext.CreateCtx
	createCtx.TemplateName = "basic"
	createCtx.TemplateSearchPaths = []string{dstDir, filepath.Join(workDir2, "templates")}
	templateCtx := NewTemplateContext()
	templateCtx.AppPath = filepath.Join(dstDir, "app1")

	// CopyAppTemplate must copy "src" template from workdir2 to workdir1 using "app1" as dst name.
	copyAppTemplate := CopyAppTemplate{}
	require.Nil(t, copyAppTemplate.Run(&createCtx, &templateCtx))
	require.DirExists(t, templateCtx.AppPath)
	require.FileExists(t, filepath.Join(templateCtx.AppPath, "init.lua"))
	require.FileExists(t, filepath.Join(templateCtx.AppPath, subdirName, "file.txt"))
}

func TestCopyTemplateDirectoryRelative(t *testing.T) {
	dstDir, err := ioutil.TempDir("", testWorkDirName)
	require.NoError(t, err)
	defer os.RemoveAll(dstDir)

	workDir2, err := ioutil.TempDir("", testWorkDirName)
	require.NoError(t, err)
	defer os.RemoveAll(workDir2)

	require.Nil(t, copy.Copy("testdata/copy_template", workDir2))

	var createCtx cmdcontext.CreateCtx
	createCtx.TemplateName = "basic"
	createCtx.TemplateSearchPaths = []string{"./templates"}
	createCtx.ConfigLocation = workDir2
	templateCtx := NewTemplateContext()
	templateCtx.AppPath = filepath.Join(dstDir, "app1")

	// CopyAppTemplate must copy "src" template from workdir2 to workdir1 using "app1" as dst name.
	copyAppTemplate := CopyAppTemplate{}
	require.Nil(t, copyAppTemplate.Run(&createCtx, &templateCtx))
	require.DirExists(t, templateCtx.AppPath)
	require.FileExists(t, filepath.Join(templateCtx.AppPath, "init.lua"))
	require.FileExists(t, filepath.Join(templateCtx.AppPath, subdirName, "file.txt"))
}

func TestExtractTemplateArchive(t *testing.T) {
	var createCtx cmdcontext.CreateCtx
	templateCtx := NewTemplateContext()

	dstDir, err := ioutil.TempDir("", testWorkDirName)
	require.NoError(t, err)
	defer os.RemoveAll(dstDir)

	workDir, err := ioutil.TempDir("", testWorkDirName)
	require.NoError(t, err)
	defer os.RemoveAll(workDir)

	srcDir := filepath.Join(workDir, "src")
	require.NoError(t, os.Mkdir(srcDir, 0755))

	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("text"), 0644))

	createCtx.TemplateSearchPaths = []string{workDir}
	templateCtx.AppPath = filepath.Join(dstDir, "app1")
	createCtx.TemplateName = "tmpl"

	archivePath := filepath.Join(workDir, "tmpl.tgz")
	archiveOut, err := os.Create(archivePath)
	require.NoError(t, err)
	defer archiveOut.Close()

	require.NoError(t, createArchive(archiveOut, filepath.Join(srcDir, "file1.txt")))

	copyAppTemplate := CopyAppTemplate{}
	require.NoError(t, copyAppTemplate.Run(&createCtx, &templateCtx))
	assert.DirExists(t, templateCtx.AppPath)
	assert.FileExists(t, filepath.Join(templateCtx.AppPath, "file1.txt"))
}