package testing

import (
	"os"
	"path"
	"runtime"
)

func init() {
	// the purpose of this is to cd to the root of the project when do testing
	// usage is
	//
	//   in some_test.go,
	//   import (
	//     _ ""liyu1981.xyz/iot-metrics-service/pkg/testing"
	//   )

	_, filename, _, _ := runtime.Caller(0)           // here runtime will return current file path
	dir := path.Join(path.Dir(filename), "..", "..") // and by double .. we will go to the project root
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}
