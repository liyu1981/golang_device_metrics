package common

import (
	"os"
	"testing"
)

func IsTestEnv() bool {
	return testing.Testing()
}
func IsDevelopment() bool {
	return os.Getenv(EnvKeyGoEnv) == "development"
}

func IsProduction() bool {
	return os.Getenv(EnvKeyGoEnv) == "production"
}

func Mapper[T any, R any](items []T, mapFn func(T) R) []R {
	mapped := make([]R, len(items))
	for i := range len(items) {
		mapped[i] = mapFn(items[i])
	}
	return mapped
}

func Reducer[T any, R any](items []T, reduceFn func(R, T) R, initAcc R) R {
	finalAcc := initAcc
	for i := range len(items) {
		finalAcc = reduceFn(finalAcc, items[i])
	}
	return finalAcc
}
