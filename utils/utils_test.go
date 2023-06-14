package utils

import "testing"

func BenchmarkFindEmpty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		find("testdata/empty", ".proto")
	}
}
func BenchmarkFindSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		find("testdata/small", ".proto")
	}
}

func BenchmarkFindLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		find("testdata/large", ".proto")
	}
}

func BenchmarkLoadLocalProtoFilesEmpty(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LoadLocalProtoFiles("testdata/empty")
	}
}

func BenchmarkLoadLocalProtoFilesSmall(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LoadLocalProtoFiles("testdata/small")
	}
}

func BenchmarkLoadLocalProtoFilesLarge(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LoadLocalProtoFiles("testdata/large")
	}
}
