package vfs

import (
	"fmt"
	"strconv"
	"strings"
)

// A Check tells if a DataProvider has a specific property or not
type Check struct {
	Test        func(dp DataProvider) error
	Name        string
	Description string
}

// A CheckResult connects a Check and its execution result.
type CheckResult struct {
	Check  *Check
	Result error
}

type CTSResult []*CheckResult

// String returns a markdown representation of this result
func (c CTSResult) String() string {
	sb := &strings.Builder{}
	sb.WriteString("| CTS Check     | Result        |\n")
	sb.WriteString("| ------------- | ------------- |\n")
	for _, check := range c {
		sb.WriteString("| ")
		sb.WriteString(check.Check.Name)
		sb.WriteString("|")
		if check.Result != nil {
			sb.WriteString(":heavy_exclamation_mark:")
		} else {
			sb.WriteString(":white_check_mark: ")
		}
		sb.WriteString("|\n")
	}

	return sb.String()
}

type CTS struct {
	checks []*Check
}

func (t *CTS) setup() {
	t.checks = []*Check{
		isEmpty,
		canWrite0,
	}
}

func (t *CTS) Run(dp DataProvider) CTSResult {
	res := make([]*CheckResult, 0)
	t.setup()
	for _, check := range t.checks {
		err := check.Test(dp)
		res = append(res, &CheckResult{check, err})
	}
	return res
}

func generateTestSlice(len int) []byte {
	tmp := make([]byte, len)
	for i := 0; i < len; i++ {
		tmp[i] = byte(i)
	}
	return tmp
}

//======== our actual checks =============
var isEmpty = &Check{
	Test: func(dp DataProvider) error {
		list, err := ReadDir(dp, "")
		if err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}
		//not empty, try to clear to make test a bit more robust
		for _, entry := range list {
			err := dp.Delete(Path(entry.Name))
			if err != nil {
				return err
			}
		}
		// recheck
		list, err = ReadDir(dp, "")
		if err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}
		return fmt.Errorf("DataProvider is not empty and cannot clear it")
	},
	Name:        "Empty DataProvider",
	Description: "Checks the corner case of an empty DataProvider",
}
var canWrite0 = &Check{
	Test: func(dp DataProvider) error {
		paths := []Path{"", "/", "/canWrite0", "/canWrite0/subfolder", "canWrite0_1/subfolder1/subfolder2"}
		lengths := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 512, 1024, 4096, 4097, 8192, 8193}
		for _, path := range paths {
			for _, testLen := range lengths {
				tmp := generateTestSlice(testLen)
				writer, err := dp.Write(path.Child(strconv.Itoa(testLen) + ".bin"))
				if err != nil {
					return err
				}
				n, err := writer.Write(tmp)
				if err != nil {
					writer.Close()
					return err
				}

				err = writer.Close()
				if err != nil {
					return err
				}

				if n != len(tmp) {
					return fmt.Errorf("expected to write %v bytes but just wrote %v", len(tmp), n)
				}
			}
		}

		return nil
	},
	Name:        "A simple write test",
	Description: "Write some simple files with various lengths in various paths",
}
