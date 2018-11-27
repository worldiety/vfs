package vfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func createTmpDir(t *testing.T) Path {
	strPath, err := ioutil.TempDir("", "wdy_vfs_test")
	if err != nil {
		t.Fatal("no tmp fs", err)
	}
	err = os.MkdirAll(strPath, os.ModePerm)
	if err != nil {
		t.Fatal("unable to create tmp dir", err)
	}
	return Path(strPath)
}

func TestEmpty(t *testing.T) {
	path := createTmpDir(t)
	fs := &FilesystemDataProvider{}
	dir, err := fs.ReadDir(path)
	if err != nil {
		t.Fatal("unexpected read failure", err)
	}
	if dir.Size() != 0 {
		t.Fatal("expected empty dir but got", dir.Size())
	}

	infos, err := ReadDir(fs, path)
	if err != nil {
		t.Fatal("unexpected read failure", err)
	}
	if len(infos) != 0 {
		t.Fatal("expected empty dir but got", dir.Size())
	}
}

func TestCTS(t *testing.T) {
	path := createTmpDir(t)
	fs := &FilesystemDataProvider{Prefix: path.String()}

	cts := &CTS{}
	cts.All()
	result := cts.Run(fs)
	fmt.Printf("\n\n%v\n\n", result.String())
	for _, check := range result {
		if check.Result != nil {
			t.Fatal(check.Check.Name, "failed:", reflect.TypeOf(check.Result), ":", check.Result)
		}
	}

}
func TestFiles(t *testing.T) {

	fileSets := [][]*testFile{
		{{"file0.bin", []byte{1}}},
		{{"file0.bin", []byte{1}}, {"file1.bin", []byte{1, 2}}},
		{{"file0.bin", []byte{1}}, {"file1.bin", []byte{1, 2}}, {"file2.bin", []byte{1, 2, 3}}},
		{{"file0.bin", []byte{1}}, {"file1.bin", []byte{1, 2}}, {"file2.bin", []byte{1, 2, 3}}, {"file3.bin", []byte{1, 2, 3, 4}}},
	}

	modes := []testMode{testModeNormal, testModePrefix}

	for _, mode := range modes {
		for _, fileSet := range fileSets {
			var path Path
			var fs *FilesystemDataProvider
			switch mode {
			case testModeNormal:
				path = createTmpDir(t)
				fs = &FilesystemDataProvider{}
			case testModePrefix:
				absPath := createTmpDir(t)
				path = ""
				fs = &FilesystemDataProvider{Prefix: absPath.String()}
			}

			for _, tf := range fileSet {
				filePath := path.Child(tf.name)
				writer, err := fs.Write(filePath)
				if err != nil {
					t.Fatal("unexpected write failure", err)
				}
				n, err := writer.Write(tf.data)
				if n != len(tf.data) {
					t.Fatal("expected to write ", len(tf.data), "but got", n)
				}
				if err != nil {
					t.Fatal("unexpected error", err)
				}
				err = writer.Close()
				if err != nil {
					t.Fatal("unexpected error", err)
				}
			}

			dir, err := fs.ReadDir(path)
			if err != nil {
				t.Fatal("unexpected read failure", err)
			}
			if int(dir.Size()) != len(fileSet) {
				t.Fatal("expected", len(fileSet), " in dir but got", dir.Size())
			}

			infos, err := ReadDir(fs, path)
			if err != nil {
				t.Fatal("unexpected read failure", err)
			}
			if len(infos) != len(fileSet) {
				t.Fatal("expected", len(fileSet), " in dir but got", len(infos))
			}

			//read files
			for _, info := range infos {
				src := byName(t, fileSet, info.Name)
				if len(src.data) != int(info.Size) {
					t.Fatal("expected ", len(src.data), "bytes but got", info.Size)
				}
				if !info.Mode.IsRegular() {
					t.Fatal("expected file, but got", info.Mode.String())
				}
				if info.Mode.IsDir() {
					t.Fatal("expected not a dir", info.Mode.String())
				}

				reader, err := fs.Read(path.Child(src.name))
				if err != nil {
					t.Fatal("expected to read file", err)
				}
				tmp := make([]byte, len(src.data))
				n, err := reader.Read(tmp)
				if err != nil {
					t.Fatal("expected to read file", err)
				}
				if n != len(src.data) {
					t.Fatal("expected to read", len(src.data), "bytes but got", n)
				}
				reader.Close()
			}

			//delete files
			for _, info := range infos {
				err := fs.Delete(path.Child(info.Name))
				if err != nil {
					t.Fatal("expected to delete file", err)
				}
			}

			infos, err = ReadDir(fs, path)
			if err != nil {
				t.Fatal("unexpected read failure", err)
			}
			if len(infos) != 0 {
				t.Fatal("expected 0 files in dir but got", len(infos))
			}
		}
	}
}

type testMode int

const testModeNormal testMode = 0
const testModePrefix testMode = 1

type testFile struct {
	name string
	data []byte
}

func byName(t *testing.T, files []*testFile, name string) *testFile {
	for _, f := range files {
		if f.name == name {
			return f
		}
	}
	t.Fatal("expected file", name)
	return nil
}
