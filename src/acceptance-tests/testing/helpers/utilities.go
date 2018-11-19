package helpers

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

func WriteFile(data string) (string, error) {
	tempFile, err := ioutil.TempFile("", "testfile")
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(tempFile.Name(), []byte(data), os.ModePerm)
	if err != nil {
		return "", err
	}

	return tempFile.Name(), nil
}

func CreateTempDir() (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	return dir, nil
}

func SetPermissions(pathToFile string, mode os.FileMode) error {
	if err := os.Chmod(pathToFile, mode); err != nil {
		return err
	}
	return nil
}

func GetUUID() string {
	guid := "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

	b := make([]byte, 16)
	_, err := rand.Read(b[:])
	if err == nil {
		guid = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	}
	return guid
}

func RunCommand(cmd *exec.Cmd) (string, string, error) {
	var stdout, stderr string

	out, err := cmd.Output()

	stdout = string(out)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			stderr = string(exitError.Stderr)
		}
	}

	return stdout, stderr, err
}
