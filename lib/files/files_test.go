package files

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifySecurity_Valid(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	uid := uint32(os.Getuid())
	os.Chmod(tmpFile.Name(), 0600)

	err = VerifySecurity(tmpFile.Name(), uid, 0600)
	assert.NoError(t, err)
}

func TestVerifySecurity_MissingFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_nonexistent_*")
	if err != nil {
		t.Fatal(err)
	}
	nonexistentPath := tmpFile.Name()
	tmpFile.Close()
	os.Remove(nonexistentPath)

	err = VerifySecurity(nonexistentPath, 0, 0600)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingFile)
}

func TestVerifySecurity_WrongUid(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	os.Chmod(tmpFile.Name(), 0600)

	err = VerifySecurity(tmpFile.Name(), 9999, 0600)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNotOwnedByUid)
}

func TestVerifySecurity_InsecurePermissions(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	os.Chmod(tmpFile.Name(), 0777)

	uid := uint32(os.Getuid())
	err = VerifySecurity(tmpFile.Name(), uid, 0600)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInsecurePerms)
}

func TestVerifySecurity_Symlink(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tmpLink := tmpFile.Name() + ".link"
	os.Symlink(tmpFile.Name(), tmpLink)
	defer os.Remove(tmpLink)

	uid := uint32(os.Getuid())
	err = VerifySecurity(tmpLink, uid, 0600)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrIsSymlink)
}
