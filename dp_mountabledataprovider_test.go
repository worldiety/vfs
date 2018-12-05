package vfs

import "testing"

func TestSimpleDelegation(t *testing.T) {
	path := createTmpDir(t)
	fs := &FilesystemDataProvider{Prefix: path.String()}

	dp := &MountableDataProvider{}
	dp.Mount("mnt/local", fs)
	SetDefault(dp)

	infos, err := ReadDirEnt("/")
	if err != nil {
		t.Fatalf("%v", err)
	}

	if len(infos) != 1 {
		t.Fatalf("expected 1 entries but got %v", len(infos))
	}

	//
	infos, err = ReadDirEnt("/mnt/")
	if err != nil {
		t.Fatalf("%v", err)
	}

	if len(infos) != 1 {
		t.Fatalf("expected 1 entries but got %v", len(infos))
	}

	//
	infos, err = ReadDirEnt("/mnt/local")
	if err != nil {
		t.Fatal(err)
	}

	if len(infos) != 0 {
		t.Fatalf("expected 0 entries but got %v", len(infos))
	}

	// write into mounted dir
	c := Path("/mnt/local/c.bin")
	_, err = WriteAll(c, generateTestSlice(13))
	if err != nil {
		t.Fatal(err)
	}

	// read from mounted dir
	data, err := ReadAll(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 13 {
		t.Fatalf("expected 13 bytes but got %v", len(data))
	}

	// stat from mounted dir
	stat, err := Stat(c)
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size != 13 {
		t.Fatalf("expected 13 bytes but got %v", stat)
	}

	// list again
	infos, err = ReadDirEnt("/mnt/local")
	if err != nil {
		t.Fatal(err)
	}

	if len(infos) != 1 {
		t.Fatalf("expected 1 entries but got %v", len(infos))
	}

	// rename
	d := Path("/mnt/local/d.bin")
	err = dp.Rename(c, d)
	if err != nil {
		t.Fatal(err)
	}

	// delete
	err = dp.Delete(d)
	if err != nil {
		t.Fatal(err)
	}

	// list again
	infos, err = ReadDirEnt("/mnt/local")
	if err != nil {
		t.Fatal(err)
	}

	if len(infos) != 0 {
		t.Fatalf("expected 0 entries but got %v", len(infos))
	}

	// mkdirs
	e := Path("/mnt/local/x/y/z")
	err = dp.MkDirs(e)
	if err != nil {
		t.Fatal(err)
	}
	stat, err = Stat(e)
	if err != nil {
		t.Fatal(err)
	}

	// unmount
	err = dp.Delete("/mnt/local")
	if err != nil {
		t.Fatal(err)
	}

	// check
	stat, err = Stat(e)
	if err == nil {
		t.Fatal("expected error but got success")
	}
}
